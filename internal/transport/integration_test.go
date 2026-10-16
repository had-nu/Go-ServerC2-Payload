package transport_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/had-nu/Go-ServerC2-Payload/internal/agentstore"
	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
	"github.com/had-nu/Go-ServerC2-Payload/internal/transport"
)

const (
	secret   = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	agentID  = "e51ec4c6-a873-4f01-9a2b-0c1d2e3f4a5b"
	hostname = "ubuntu-target"
)

type harness struct {
	t      *testing.T
	server *httptest.Server
	store  *agentstore.Store
	events chan transport.Event
	client *http.Client
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	store := agentstore.New()
	events := make(chan transport.Event, 16)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := transport.NewServer(store, secret, log, events)

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	return &harness{
		t:      t,
		server: ts,
		store:  store,
		events: events,
		client: ts.Client(),
	}
}

func (h *harness) do(method, path, body string) *http.Response {
	h.t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, h.server.URL+path, r)
	if err != nil {
		h.t.Fatalf("new request: %v", err)
	}
	req.Header.Set(c2proto.HeaderAuth, "Bearer "+secret)
	req.Header.Set(c2proto.HeaderAgentID, agentID)
	req.Header.Set(c2proto.HeaderHostname, hostname)
	resp, err := h.client.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func readJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

// A Beacon checks in, receives nothing, the operator queues a task, the Beacon
// checks in again and receives it, and it posts the result.
func TestEndToEndTaskLifecycle(t *testing.T) {
	h := newHarness(t)

	if resp := h.do(http.MethodGet, "/checkin", ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("first check-in = %d, want %d", resp.StatusCode, http.StatusNoContent)
	} else {
		resp.Body.Close()
	}
	if h.store.Len() != 1 {
		t.Fatalf("store.Len() = %d, want 1", h.store.Len())
	}
	<-h.events // new agent

	task := c2proto.Task{TaskID: "1709566971000000000", Command: "id"}
	if err := h.store.Enqueue(agentID, task); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	resp := h.do(http.MethodGet, "/checkin", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second check-in = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	got := readJSON[c2proto.Task](t, resp)
	if got != task {
		t.Fatalf("task = %+v, want %+v", got, task)
	}

	body, _ := json.Marshal(c2proto.Result{TaskID: task.TaskID, Output: "uid=1000(user)\n"})
	resp2 := h.do(http.MethodPost, "/results", string(body))
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("results = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	ev := <-h.events
	if ev.Kind != transport.EventResult {
		t.Fatalf("event kind = %v, want EventResult", ev.Kind)
	}
	if !strings.Contains(ev.Text, "uid=1000") {
		t.Fatalf("event text = %q, want the output", ev.Text)
	}
}

func TestEndToEndTaskQueueIsOrdered(t *testing.T) {
	h := newHarness(t)
	resp := h.do(http.MethodGet, "/checkin", "")
	resp.Body.Close()
	<-h.events

	want := []c2proto.Task{
		{TaskID: "1709566971000000000", Command: "id"},
		{TaskID: "1709566971000000001", Command: "hostname"},
		{TaskID: "1709566971000000002", Command: "uname -a"},
	}
	for _, task := range want {
		if err := h.store.Enqueue(agentID, task); err != nil {
			t.Fatalf("Enqueue %s: %v", task.TaskID, err)
		}
	}
	for i, w := range want {
		r := h.do(http.MethodGet, "/checkin", "")
		got := readJSON[c2proto.Task](t, r)
		if got != w {
			t.Fatalf("task %d = %+v, want %+v", i, got, w)
		}
	}
	r := h.do(http.MethodGet, "/checkin", "")
	defer r.Body.Close()
	if r.StatusCode != http.StatusNoContent {
		t.Fatalf("the queue must be empty, got %d", r.StatusCode)
	}
}

// A client with no secret must never reach the store, whatever it sends.
func TestEndToEndUnauthorizedNeverReachesTheStore(t *testing.T) {
	h := newHarness(t)

	attempts := []struct{ method, path, body string }{
		{http.MethodGet, "/checkin", ""},
		{http.MethodPost, "/results", `{"task_id":"1709566971000000000","output":"x"}`},
	}
	for _, a := range attempts {
		req, err := http.NewRequest(a.method, h.server.URL+a.path, strings.NewReader(a.body))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set(c2proto.HeaderAgentID, agentID)
		resp, err := h.client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", a.method, a.path, err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s = %d, want %d", a.method, a.path, resp.StatusCode, http.StatusUnauthorized)
		}
		resp.Body.Close()
	}
	if h.store.Len() != 0 {
		t.Fatalf("an unauthorized request registered %d agents", h.store.Len())
	}
}

func TestEndToEndShortAgentIDDoesNotPanic(t *testing.T) {
	h := newHarness(t)

	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/checkin", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(c2proto.HeaderAuth, "Bearer "+secret)
	req.Header.Set(c2proto.HeaderAgentID, "ab")
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("check-in: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	if h.store.Len() != 0 {
		t.Fatal("a rejected agent ID was stored")
	}

	// The console rendered this value with ID[:8]. Render it now.
	if got := c2proto.ShortID("ab"); got != "ab" {
		t.Fatalf("ShortID = %q, want %q", got, "ab")
	}
}

// A hostile hostname must reach the store truncated and without control
// characters.
//
// The Go client refuses to send control characters in a header, so a test that
// uses it cannot inject them. SanitizeHeader covers that case on its own in
// proto_test.go. This test sends what a non-Go client can send: a very long
// value.
func TestEndToEndLongHostnameIsTruncated(t *testing.T) {
	store := agentstore.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ts := httptest.NewServer(transport.NewServer(store, secret, log, nil))
	defer ts.Close()

	const hostHeaderLimit = 3000
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/checkin", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(c2proto.HeaderAuth, "Bearer "+secret)
	req.Header.Set(c2proto.HeaderAgentID, agentID)
	req.Header.Set(c2proto.HeaderHostname, strings.Repeat("a", hostHeaderLimit))

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("check-in: %v", err)
	}
	defer resp.Body.Close()

	snap := store.Snapshot(agentstore.DefaultTTL)
	if len(snap) != 1 {
		t.Fatalf("want 1 agent, got %d", len(snap))
	}
	if len(snap[0].Hostname) >= hostHeaderLimit {
		t.Fatalf("the stored hostname has %d bytes, the handler must truncate it", len(snap[0].Hostname))
	}
	if !strings.HasSuffix(snap[0].Hostname, "...") {
		t.Fatalf("the stored hostname was not marked as truncated: %q", snap[0].Hostname[len(snap[0].Hostname)-6:])
	}
}

func TestEndToEndOverTLS(t *testing.T) {
	store := agentstore.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cert, err := transport.LoadOrCreateCert(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("LoadOrCreateCert: %v", err)
	}

	srv := transport.HTTPServer("127.0.0.1:0", transport.NewServer(store, secret, log, nil), cert)
	ln, err := transport.ListenTLS(srv)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()
	defer transport.Shutdown(srv)

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, // #nosec G402 -- test only
	}}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/checkin", ln.Addr()), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(c2proto.HeaderAuth, "Bearer "+secret)
	req.Header.Set(c2proto.HeaderAgentID, agentID)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("check-in over TLS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if resp.TLS == nil {
		t.Fatal("the connection did not use TLS")
	}
	if store.Len() != 1 {
		t.Fatalf("store.Len() = %d, want 1", store.Len())
	}
}

func TestEndToEndConcurrentBeacons(t *testing.T) {
	h := newHarness(t)

	const beacons = 12
	done := make(chan struct{}, beacons)
	for i := range beacons {
		id := fmt.Sprintf("agent-%08d", i)
		go func() {
			defer func() { done <- struct{}{} }()
			for range 5 {
				req, err := http.NewRequest(http.MethodGet, h.server.URL+"/checkin", nil)
				if err != nil {
					return
				}
				req.Header.Set(c2proto.HeaderAuth, "Bearer "+secret)
				req.Header.Set(c2proto.HeaderAgentID, id)
				resp, err := h.client.Do(req)
				if err != nil {
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}
	for range beacons {
		<-done
	}

	if got := h.store.Len(); got != beacons {
		t.Fatalf("store.Len() = %d, want %d", got, beacons)
	}
}

var _ = bytes.MinRead
