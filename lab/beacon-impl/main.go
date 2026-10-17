//go:build !lab

// Command beacon builds as an inert program unless the build tag "lab" is set.
//
// This file holds the default build. A reader who runs "go build ./..." gets
// this program. It names the steps and stops. It runs nothing.
//
// The working Lab Beacon is in beacon_lab.go, behind the build tag. Build it
// with:
//
//	go build -tags lab ./lab/beacon-impl
//
// The Makefile target "make lab-beacon" runs that command.
//
// The working Beacon also needs the flag -i-run-remote-commands before it runs
// a command. Both gates exist so that a reader must take two deliberate acts
// before a command runs on any machine.
//
// See docs/LAB.md for the full procedure and for the purpose of each step.
package main

import (
	"fmt"
	"os"
	"runtime"
)

const instructions = `beacon: this build is inert.

This binary runs no command. The default build has no beacon loop, no HTTP
client, and no call to exec.

To build the working Lab Beacon:

    go build -tags lab ./lab/beacon-impl

The Makefile target does the same thing:

    make lab-beacon

To run the working Lab Beacon, the operator must also pass this flag:

    -i-run-remote-commands

The flag is the second gate. A binary built with the tag but started without
the flag checks in, registers, and refuses every task.

Read docs/LAB.md before you continue. It states what each step does and why.`

func main() {
	fmt.Println(instructions)
	fmt.Println()
	fmt.Printf("beacon: go version %s, platform %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("beacon: the binary is %d bytes and has no network code\n", sizeOfThisBinary())
	_ = os.Stdout
}

// sizeOfThisBinary returns the size of the running binary, or 0 when the path
// cannot be read.
func sizeOfThisBinary() int64 {
	path, err := os.Executable()
	if err != nil {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
