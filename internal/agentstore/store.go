// Package agentstore keeps the Team Server state: the registered Beacons and
// the task queue of each Beacon. The state is in a Store value, not in package
// variables, so that tests can build one.
package agentstore

import (
	"errors"
	"net/netip"
	"sync"
	"time"

	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
)

// DefaultTTL is the time without a check-in before the console shows an agent
// as dead.
const DefaultTTL = 30 * time.Second

// ErrNotFound means that no agent uses the given ID.
var ErrNotFound = errors.New("agent not found")

// ErrQueueFull means that the agent has too many tasks waiting.
var ErrQueueFull = errors.New("task queue is full")

// MaxQueue is the largest number of tasks the store keeps for one agent.
const MaxQueue = 64

// Agent is one registered Beacon. The task queue is private to the Store.
type Agent struct {
	ID       string
	Hostname string
	IP       netip.Addr
	LastSeen time.Time
}

// Info is a copy of an agent for display in the console.
type Info struct {
	ID       string
	ShortID  string
	Hostname string
	IP       string
	LastSeen time.Time
	Age      time.Duration
	Pending  int
	Alive    bool
}

// Store holds the agent table. A Store is safe for concurrent use.
type Store struct {
	mu     sync.Mutex
	agents map[string]*agent
	queues map[string][]c2proto.Task
	now    func() time.Time
}

type agent struct {
	Agent
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		agents: make(map[string]*agent),
		queues: make(map[string][]c2proto.Task),
		now:    time.Now,
	}
}

// SetClock replaces the time source. Tests use it to control age and liveness.
func (s *Store) SetClock(f func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = f
}

// Checkin registers an agent or refreshes it, and pops one task if the queue is
// not empty. It returns the task and true when the agent is new.
//
// The caller receives the task under no lock, so it can write the response
// without blocking other agents.
func (s *Store) Checkin(id, hostname string, ip netip.Addr) (task *c2proto.Task, isNew bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, exists := s.agents[id]
	if !exists {
		a = &agent{Agent: Agent{ID: id, Hostname: hostname, IP: ip}}
		s.agents[id] = a
		s.queues[id] = nil
		isNew = true
	}
	if hostname != "" {
		a.Hostname = hostname
	}
	if ip.IsValid() {
		a.IP = ip
	}
	a.LastSeen = s.now()

	queue := s.queues[id]
	if len(queue) == 0 {
		return nil, isNew
	}
	popped := queue[0]
	s.queues[id] = queue[1:]
	return &popped, isNew
}

// Touch refreshes the last-seen time of an agent.
func (s *Store) Touch(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return false
	}
	a.LastSeen = s.now()
	return true
}

// Enqueue adds a task to the queue of an agent.
func (s *Store) Enqueue(id string, task c2proto.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agents[id]; !ok {
		return ErrNotFound
	}
	if len(s.queues[id]) >= MaxQueue {
		return ErrQueueFull
	}
	s.queues[id] = append(s.queues[id], task)
	return nil
}

// Pending returns the number of tasks waiting for an agent.
func (s *Store) Pending(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queues[id])
}

// Snapshot returns a copy of the agent table, sorted by ID so that the console
// output is stable between calls.
func (s *Store) Snapshot(ttl time.Duration) []Info {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	out := make([]Info, 0, len(s.agents))
	for id, a := range s.agents {
		age := now.Sub(a.LastSeen)
		out = append(out, Info{
			ID:       id,
			ShortID:  c2proto.ShortID(id),
			Hostname: a.Hostname,
			IP:       a.IP.String(),
			LastSeen: a.LastSeen,
			Age:      age.Round(time.Second),
			Pending:  len(s.queues[id]),
			Alive:    age <= ttl,
		})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Exists reports whether an agent is registered.
func (s *Store) Exists(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.agents[id]
	return ok
}

// Len returns the number of registered agents.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.agents)
}
