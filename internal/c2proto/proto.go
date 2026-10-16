// Package c2proto defines the wire format that the Team Server and the Beacon
// share. Both programs import this package, so each type has exactly one
// definition.
package c2proto

import (
	"errors"
	"fmt"
	"strings"
)

// HTTP headers used by both programs.
const (
	HeaderAgentID  = "X-Agent-ID"
	HeaderHostname = "X-Hostname"
	HeaderAuth     = "Authorization"
)

// TaskIDLen is the exact length of a task identifier.
const TaskIDLen = 19

// Agent ID limits. The lower bound removes a crash in the Team Server: the
// operator console used to slice the first 8 bytes of the agent ID.
const (
	MinAgentIDLen = 8
	MaxAgentIDLen = 64
)

// MaxCommandLen is the largest command that the Team Server accepts.
const MaxCommandLen = 4096

// Errors returned by the validators.
var (
	ErrAgentIDTooShort = fmt.Errorf("agent ID has fewer than %d characters", MinAgentIDLen)
	ErrAgentIDTooLong  = fmt.Errorf("agent ID has more than %d characters", MaxAgentIDLen)
	ErrAgentIDChar     = errors.New("agent ID contains a character outside the allowed set")
	ErrCommandTooLong  = fmt.Errorf("command is longer than %d bytes", MaxCommandLen)
	ErrTaskIDLen       = fmt.Errorf("task ID is not %d characters", TaskIDLen)
)

// Task is a command that the Team Server queues for one Beacon.
type Task struct {
	TaskID  string `json:"task_id"`
	Command string `json:"command"`
}

// Result is the outcome of one executed Task.
type Result struct {
	TaskID string `json:"task_id"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

// IsError reports whether the Beacon recorded a failure.
func (r Result) IsError() bool { return r.Error != "" }

// ValidateAgentID checks an agent ID before the Team Server stores it. The
// Beacon builds the ID, but any HTTP client can send any value.
func ValidateAgentID(id string) error {
	if len(id) < MinAgentIDLen {
		return ErrAgentIDTooShort
	}
	if len(id) > MaxAgentIDLen {
		return ErrAgentIDTooLong
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("%w: %q", ErrAgentIDChar, r)
		}
	}
	return nil
}

// ValidateCommand checks a command before the Team Server queues it.
func ValidateCommand(cmd string) error {
	if len(cmd) > MaxCommandLen {
		return ErrCommandTooLong
	}
	return nil
}

// ValidateTaskID checks a task identifier before the Team Server queues it.
func ValidateTaskID(id string) error {
	if len(id) != TaskIDLen {
		return ErrTaskIDLen
	}
	return nil
}

// ShortID returns the first 8 characters of an ID. It never panics, because it
// returns the whole ID when the ID is shorter.
func ShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// SanitizeHeader makes an HTTP header value safe to write to a log. It removes
// control characters and caps the length.
func SanitizeHeader(v string, max int) string {
	var b strings.Builder
	b.Grow(min(len(v), max))
	for _, r := range v {
		if r < 0x20 || r == 0x7f {
			b.WriteByte(' ')
			continue
		}
		if b.Len() >= max {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}
