//go:build linux

package opslinux

import (
	"errors"
	"strings"
	"testing"
)

func TestSetProcessNameRoundTrip(t *testing.T) {
	const want = "[kworker/u4:2]"

	before, err := ProcessName()
	if err != nil {
		t.Fatalf("ProcessName before: %v", err)
	}
	t.Cleanup(func() {
		if err := SetProcessName(before); err != nil {
			t.Errorf("restore the process name: %v", err)
		}
	})

	if err := SetProcessName(want); err != nil {
		t.Fatalf("SetProcessName(%q): %v", want, err)
	}
	got, err := ProcessName()
	if err != nil {
		t.Fatalf("ProcessName after: %v", err)
	}
	if got != want {
		t.Fatalf("ProcessName() = %q, want %q", got, want)
	}
}

// The kernel limits comm to 15 bytes plus a null terminator.
func TestSetProcessNameLengthLimit(t *testing.T) {
	longest := strings.Repeat("a", MaxCommLen-1)

	if err := SetProcessName(longest); err != nil {
		t.Fatalf("SetProcessName at %d bytes: %v", MaxCommLen-1, err)
	}
	if err := SetProcessName(strings.Repeat("a", MaxCommLen)); !errors.Is(err, ErrNameTooLong) {
		t.Fatalf("SetProcessName at %d bytes = %v, want ErrNameTooLong", MaxCommLen, err)
	}
	if err := SetProcessName(strings.Repeat("a", 64)); !errors.Is(err, ErrNameTooLong) {
		t.Fatalf("SetProcessName at 64 bytes = %v, want ErrNameTooLong", err)
	}
}

func TestSetProcessNameRejectsBadInput(t *testing.T) {
	if err := SetProcessName(""); !errors.Is(err, ErrNameEmpty) {
		t.Fatalf("SetProcessName(\"\") = %v, want ErrNameEmpty", err)
	}
	if err := SetProcessName("kworkeré"); !errors.Is(err, ErrNameNotASCII) {
		t.Fatalf("SetProcessName(non-ASCII) = %v, want ErrNameNotASCII", err)
	}
}

// The name in the README must fit in the kernel field.
func TestKernelWorkerNameFits(t *testing.T) {
	const name = "[kworker/u4:2]"
	if len(name) > MaxCommLen-1 {
		t.Fatalf("%q has %d bytes, the kernel allows %d", name, len(name), MaxCommLen-1)
	}
	if err := SetProcessName(name); err != nil {
		t.Fatalf("SetProcessName(%q): %v", name, err)
	}
}
