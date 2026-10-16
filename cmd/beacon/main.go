// Command beacon is the published interface of the Beacon, the agent side of
// the C2 teaching kit.
//
// This program does not run tasks. It is the documented shape of the Beacon:
// the same functions, the same order of calls, and no behaviour. A reader can
// see how a Beacon is structured without a working implant in the repository.
//
// The functions return zero values. Every body is empty on purpose.
//
// The working Beacon lives in lab/beacon-impl, which the .gitignore file
// excludes. A lab operator builds it with "make lab-beacon".
package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// Configuration. A working Beacon takes these values from flags or from a
// configuration file, not from constants.
const (
	// DefaultServerURL is the Team Server that the Beacon calls.
	DefaultServerURL = "https://127.0.0.1:8443"

	// DefaultSleep is the base interval between two check-ins.
	DefaultSleep = 10 * time.Second

	// DefaultJitter is the largest change to the interval, in percent.
	DefaultJitter = 20

	// CheckinTimeout is the limit on one HTTP request.
	CheckinTimeout = 5 * time.Second

	// MaxResultBytes is the largest result that the Beacon sends.
	MaxResultBytes = 1 << 20

	// MaxAttempts is how many times the Beacon repeats a failed send.
	MaxAttempts = 3

	// RetryDelay is the pause between two attempts.
	RetryDelay = 2 * time.Second
)

// agentID identifies this Beacon to the Team Server. The Team Server rejects
// an ID with fewer than 8 characters.
var agentID string

// hostname is the name of the host where the Beacon runs.
var hostname string

// startTime lets the Beacon report how long it has run.
var startTime = time.Now()

func main() {
	fmt.Println("beacon: this is the interface stub, not a working agent")
	fmt.Printf("beacon: it would check in to %s every %s with %d%% jitter\n",
		DefaultServerURL, DefaultSleep, DefaultJitter)
	fmt.Printf("beacon: functions declared here: init, checkIn, postResults, runTask, setProcessName, main\n")
	fmt.Printf("beacon: go version %s, platform %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println("beacon: read docs/ARCHITECTURE.md for the sequence of calls")
	fmt.Println("beacon: read docs/DETECTION.md for what this traffic looks like")
	_ = os.Stdout
}

// init makes a unique identifier, reads the host name, and starts the random
// generator. A working Beacon calls these in main, not in init, so that a test
// can build one without the side effects of package initialisation.
func init() {}

// getSleepDuration returns the wait before the next check-in.
//
// The Beacon adds a random change to the base interval. With the default
// values the result is between 8 and 12 seconds.
//
// A working Beacon must not use math/rand.Seed. That function does nothing
// after Go 1.24.
func getSleepDuration() time.Duration { return 0 }

// checkIn calls the Team Server and returns one task.
//
// A working Beacon sends the identifier and the host name as headers, reads a
// 204 as "no task", and returns nil without an error.
func checkIn() (*Task, error) { return nil, nil }

// postResults sends the output of one task to the Team Server.
//
// A working Beacon must check the status code. The first version of this
// project ignored it, so the Beacon believed that a failed send had worked.
func postResults(result Result) error { return nil }

// runTask runs one command and sends the result back.
//
// A working Beacon gives the command to /bin/bash with the -c option, captures
// the standard output and the standard error, and sends both.
func runTask(task Task) {}

// setProcessName changes the name that ps and top show.
//
// A working Beacon calls the Linux system call prctl with PR_SET_NAME. The
// internal/opslinux package does this. The kernel allows 15 characters.
func setProcessName(name string) error { return nil }

// Task is a command that the Team Server queued for this Beacon.
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
