// Package opslinux wraps the Linux calls that this project uses. The process
// name call is the one that an analyst looks for, so it lives in its own
// package with tests.
package opslinux

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// MaxCommLen is the size of the comm field in the Linux task structure. The
// kernel stores the name with a null terminator, so the longest usable name has
// MaxCommLen-1 characters.
const MaxCommLen = 16

// Errors returned by SetProcessName.
var (
	ErrNameEmpty    = errors.New("process name is empty")
	ErrNameTooLong  = fmt.Errorf("process name is longer than %d bytes", MaxCommLen-1)
	ErrNameNotASCII = errors.New("process name contains a non-ASCII byte")
)

// SetProcessName changes the name that the ps and top commands show for the
// current process. The kernel limits the name, so the function checks the
// length before it calls the kernel.
//
// This call changes only the comm field. It does not change the command line in
// the process table, the executable path, or the parent process.
func SetProcessName(name string) error {
	if name == "" {
		return ErrNameEmpty
	}
	if len(name) > MaxCommLen-1 {
		return fmt.Errorf("%w: %d bytes", ErrNameTooLong, len(name))
	}
	for i := range len(name) {
		if name[i] >= 0x80 {
			return fmt.Errorf("%w: byte %d", ErrNameNotASCII, i)
		}
	}

	// The kernel reads a null-terminated string through this pointer.
	buf := append([]byte(name), 0)
	// #nosec G103 -- the pointer is valid for the duration of the syscall, and
	// the buffer is not moved or reallocated while the kernel reads it.
	if err := unix.Prctl(unix.PR_SET_NAME, uintptr(unsafe.Pointer(&buf[0])), 0, 0, 0); err != nil {
		return fmt.Errorf("prctl(PR_SET_NAME): %w", err)
	}
	return nil
}

// ProcessName returns the comm field of the current process. It reads the same
// kernel field that SetProcessName writes.
func ProcessName() (string, error) {
	buf := make([]byte, MaxCommLen)
	// #nosec G103 -- the buffer is a local slice of the right size for the
	// kernel field, and it stays alive for the duration of the call.
	if err := unix.Prctl(unix.PR_GET_NAME, uintptr(unsafe.Pointer(&buf[0])), 0, 0, 0); err != nil {
		return "", fmt.Errorf("prctl(PR_GET_NAME): %w", err)
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}
