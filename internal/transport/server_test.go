package transport

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/had-nu/Go-ServerC2-Payload/internal/agentstore"
	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
)

const testSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const testAgent = "e51ec4c6-a873-4f01-9a2b-0c1d2e3f4a5b"

func newTestServer(t *testing.T) (*Server, *agentstore.Store, chan Event) {
	t.Helper()
	store := agentstore.New()
	events := make(chan Event, 8)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(store, testSecret, log, events), store, events
}

func authed(method, path string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, path, body)
	r.Header.Set(c2proto.HeaderAuth, "Bearer "+testSecret)
	return r
}

// The first version of the server had no authentication at all.
func TestHandlersRequireTheSharedSecret(t *testing.T) {
	srv, _, _ := newTestServer(t)

	tests := []struct {
		name    string
		method  string
		path    string
		wantErr bool
	}{
		{"check-in without a secret", http.MethodGet, "/checkin", false},
		{"results without a secret", http.MethodPost, "/results", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			r.Header.Set(c2proto.HeaderAgentID, testAgent)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
			if store := srv.store; store.Len() != 0 {
				t.Fatalf("an unauthorized request registered %d agents", store.Len())
			}
			if got := w.Header().Get("WWW-Authenticate"); got == "" {
				t.Error("the response has no WWW-Authenticate header")
			}
		})
	}
}

func TestCheckinRejectsBadAgentID(t *testing.T) {
	srv, _, _ := newTestServer(t)

	bad := []string{"", "ab", strings.Repeat("a", c2proto.MaxAgentIDLen+1), "abcd efgh", "abcd\nefgh"}
	for _, id := range bad {
		t.Run(id, func(t *testing.T) {
			r := authed(http.MethodGet, "/checkin", nil)
			r.Header.Set(c2proto.HeaderAgentID, id)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d for agent ID %q, want %d", w.Code, id, http.StatusBadRequest)
			}
		})
	}
	if srv.store.Len() != 0 {
		t.Fatalf("a rejected agent ID registered an agent")
	}
}

func TestCheckinNoTaskReturns204(t *testing.T) {
	srv, store, events := newTestServer(t)

	r := authed(http.MethodGet, "/checkin", nil)
	r.Header.Set(c2proto.HeaderAgentID, testAgent)
	r.Header.Set(c2proto.HeaderHostname, "ubuntu-target")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if store.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", store.Len())
	}

	select {
	case ev := <-events:
		if ev.Kind != EventAgentNew {
			t.Fatalf("event kind = %v, want EventAgentNew", ev.Kind)
		}
	default:
		t.Fatal("a new agent must produce an event")
	}
}

func TestCheckinReturnsQueuedTask(t *testing.T) {
	srv, store, _ := newTestServer(t)

	r := authed(http.MethodGet, "/checkin", nil)
	r.Header.Set(c2proto.HeaderAgentID, testAgent)
	srv.ServeHTTP(httptest.NewRecorder(), r)

	task := c2proto.Task{TaskID: "1709566971000000000", Command: "id"}
	if err := store.Enqueue(testAgent, task); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	r2 := authed(http.MethodGet, "/checkin", nil)
	r2.Header.Set(c2proto.HeaderAgentID, testAgent)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r2)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got c2proto.Task
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != task {
		t.Fatalf("task = %+v, want %+v", got, task)
	}
}

func TestMethodIsEnforcedByThePattern(t *testing.T) {
	srv, _, _ := newTestServer(t)

	// Go 1.22 method patterns replace the manual checks in the handlers.
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/checkin"},
		{http.MethodGet, "/results"},
		{http.MethodDelete, "/checkin"},
	} {
		r := authed(tc.method, tc.path, strings.NewReader("{}"))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestResultsAcceptsAValidBody(t *testing.T) {
	srv, store, events := newTestServer(t)

	r := authed(http.MethodGet, "/checkin", nil)
	r.Header.Set(c2proto.HeaderAgentID, testAgent)
	srv.ServeHTTP(httptest.NewRecorder(), r)
	<-events // the new-agent event

	body, _ := json.Marshal(c2proto.Result{TaskID: "1709566971000000000", Output: "uid=1000(user)\n"})
	r2 := authed(http.MethodPost, "/results", bytes.NewReader(body))
	r2.Header.Set(c2proto.HeaderAgentID, testAgent)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r2)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	select {
	case ev := <-events:
		if ev.Kind != EventResult {
			t.Fatalf("event kind = %v, want EventResult", ev.Kind)
		}
		if !strings.Contains(ev.Text, "uid=1000") {
			t.Fatalf("event text = %q, want the output", ev.Text)
		}
	default:
		t.Fatal("a result must produce an event")
	}
	if store.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", store.Len())
	}
}

func TestResultsRejectsBadInput(t *testing.T) {
	srv, _, _ := newTestServer(t)

	r := authed(http.MethodGet, "/checkin", nil)
	r.Header.Set(c2proto.HeaderAgentID, testAgent)
	srv.ServeHTTP(httptest.NewRecorder(), r)

	tests := []struct {
		name     string
		agent    string
		body     string
		wantCode int
	}{
		{"unknown agent", "00000000-dead-4f00-8000-000000000000", `{}`, http.StatusNotFound},
		{"invalid JSON", testAgent, `not json`, http.StatusBadRequest},
		{"bad task ID", testAgent, `{"task_id":"short"}`, http.StatusBadRequest},
		{"empty body", testAgent, ``, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := authed(http.MethodPost, "/results", strings.NewReader(tc.body))
			r.Header.Set(c2proto.HeaderAgentID, tc.agent)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)
			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantCode)
			}
		})
	}
}

// The old handler called io.ReadAll with no limit.
func TestResultsLimitsTheRequestBody(t *testing.T) {
	srv, _, _ := newTestServer(t)

	r := authed(http.MethodGet, "/checkin", nil)
	r.Header.Set(c2proto.HeaderAgentID, testAgent)
	srv.ServeHTTP(httptest.NewRecorder(), r)

	// A body that is valid JSON, so the decoder reads to the limit.
	big, _ := json.Marshal(c2proto.Result{
		TaskID: "1709566971000000000",
		Output: strings.Repeat("A", MaxRequestBody+1024),
	})
	r2 := authed(http.MethodPost, "/results", bytes.NewReader(big))
	r2.Header.Set(c2proto.HeaderAgentID, testAgent)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r2)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRenderResult(t *testing.T) {
	ok := renderResult(testAgent, c2proto.Result{TaskID: "1709566971000000000", Output: "hello"})
	if !strings.Contains(ok, "hello") || !strings.Contains(ok, "e51ec4c6") {
		t.Fatalf("renderResult(ok) = %q", ok)
	}

	failed := renderResult(testAgent, c2proto.Result{TaskID: "1709566971000000000", Error: "boom"})
	if !strings.Contains(failed, "boom") || strings.Contains(failed, "hello") {
		t.Fatalf("renderResult(error) = %q", failed)
	}
}

// The console used ID[:8], which panicked on a short ID.
func TestRenderResultWithShortAgentID(t *testing.T) {
	got := renderResult("ab", c2proto.Result{TaskID: "1709566971000000000", Output: "x"})
	if !strings.Contains(got, "ab") {
		t.Fatalf("renderResult(short) = %q", got)
	}
}

func TestHTTPServerHasLimits(t *testing.T) {
	srv := HTTPServer(":0", http.NotFoundHandler(), mustPair(t))
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatalf("the server has an unset timeout: %+v", srv)
	}
	if srv.MaxHeaderBytes != MaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, MaxHeaderBytes)
	}
	if srv.TLSConfig.MinVersion != 0x0303 {
		t.Fatalf("MinVersion = %x, want TLS 1.2", srv.TLSConfig.MinVersion)
	}
}

func TestPlainHTTPServerHasLimits(t *testing.T) {
	srv := PlainHTTPServer(":0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatalf("the server has an unset timeout: %+v", srv)
	}
	if srv.TLSConfig != nil {
		t.Fatal("PlainHTTPServer must not set TLSConfig")
	}
}
