package agentstore

import (
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
)

func newTestStore(t *testing.T) (*Store, *time.Time) {
	t.Helper()
	s := New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s.SetClock(func() time.Time { return now })
	return s, &now
}

func TestCheckinRegistersOnce(t *testing.T) {
	s, _ := newTestStore(t)
	ip := netip.MustParseAddr("192.0.2.10")

	if _, isNew := s.Checkin("e51ec4c6-a873", "target-a", ip); !isNew {
		t.Fatal("first check-in must report a new agent")
	}
	if _, isNew := s.Checkin("e51ec4c6-a873", "target-a", ip); isNew {
		t.Fatal("second check-in must not report a new agent")
	}
	if got := s.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
}

// A slow HTTP write must not hold the store lock and block other agents.
func TestCheckinReturnsTaskOutsideTheLock(t *testing.T) {
	s, _ := newTestStore(t)
	ip := netip.MustParseAddr("192.0.2.10")
	s.Checkin("e51ec4c6-a873", "target-a", ip)

	if err := s.Enqueue("e51ec4c6-a873", c2proto.Task{TaskID: "t1", Command: "id"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	task, _ := s.Checkin("e51ec4c6-a873", "target-a", ip)
	if task == nil {
		t.Fatal("Checkin returned no task after Enqueue")
	}
	if task.Command != "id" {
		t.Fatalf("task.Command = %q, want %q", task.Command, "id")
	}
	if got := s.Pending("e51ec4c6-a873"); got != 0 {
		t.Fatalf("Pending() = %d after the pop, want 0", got)
	}
	if task2, _ := s.Checkin("e51ec4c6-a873", "target-a", ip); task2 != nil {
		t.Fatal("Checkin returned a second task from an empty queue")
	}
}

func TestEnqueueUnknownAgent(t *testing.T) {
	s, _ := newTestStore(t)
	err := s.Enqueue("missing-agent", c2proto.Task{TaskID: "t1", Command: "id"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Enqueue(unknown) = %v, want ErrNotFound", err)
	}
}

func TestEnqueueRespectsQueueLimit(t *testing.T) {
	s, _ := newTestStore(t)
	s.Checkin("e51ec4c6-a873", "target-a", netip.MustParseAddr("192.0.2.10"))

	for i := range MaxQueue {
		if err := s.Enqueue("e51ec4c6-a873", c2proto.Task{TaskID: "t", Command: "id"}); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}
	err := s.Enqueue("e51ec4c6-a873", c2proto.Task{TaskID: "t", Command: "id"})
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("Enqueue over the limit = %v, want ErrQueueFull", err)
	}
}

func TestSnapshotLiveness(t *testing.T) {
	s, now := newTestStore(t)
	s.Checkin("e51ec4c6-a873", "target-a", netip.MustParseAddr("192.0.2.10"))

	snap := s.Snapshot(DefaultTTL)
	if len(snap) != 1 || !snap[0].Alive {
		t.Fatalf("agent must be alive right after the check-in: %+v", snap)
	}

	*now = now.Add(DefaultTTL + time.Second)
	snap = s.Snapshot(DefaultTTL)
	if len(snap) != 1 || snap[0].Alive {
		t.Fatalf("agent must be dead after the TTL: %+v", snap)
	}
	if snap[0].Age != DefaultTTL+time.Second {
		t.Fatalf("Age = %v, want %v", snap[0].Age, DefaultTTL+time.Second)
	}
}

func TestSnapshotIsSorted(t *testing.T) {
	s, _ := newTestStore(t)
	for _, id := range []string{"ffff0000-1111", "0000ffff-2222", "aaaa0000-3333"} {
		s.Checkin(id, "host", netip.MustParseAddr("192.0.2.10"))
	}
	snap := s.Snapshot(DefaultTTL)
	for i := 1; i < len(snap); i++ {
		if snap[i-1].ID > snap[i].ID {
			t.Fatalf("Snapshot is not sorted: %q before %q", snap[i-1].ID, snap[i].ID)
		}
	}
}

func TestTouch(t *testing.T) {
	s, now := newTestStore(t)
	s.Checkin("e51ec4c6-a873", "target-a", netip.MustParseAddr("192.0.2.10"))

	*now = now.Add(time.Hour)
	if !s.Touch("e51ec4c6-a873") {
		t.Fatal("Touch must return true for a known agent")
	}
	if s.Touch("missing-agent") {
		t.Fatal("Touch must return false for an unknown agent")
	}
	if snap := s.Snapshot(DefaultTTL); !snap[0].Alive {
		t.Fatalf("Touch must refresh liveness: %+v", snap)
	}
}

func TestConcurrentCheckinAndEnqueue(t *testing.T) {
	s := New()
	const id = "e51ec4c6-a873"

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.Checkin(id, "target-a", netip.MustParseAddr("192.0.2.10"))
		}()
		go func() {
			defer wg.Done()
			s.Enqueue(id, c2proto.Task{TaskID: "t", Command: "id"})
		}()
	}
	wg.Wait()

	if got := s.Pending(id); got > MaxQueue {
		t.Fatalf("Pending() = %d, want <= %d", got, MaxQueue)
	}
}
