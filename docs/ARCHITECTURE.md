# Architecture

This document describes how the Team Server and the Beacon exchange data.

The text follows the writing style of ASD-STE100 Issue 9. The style shortens
the sentences. It does not certify the text against the standard.

## The two programs

The repository contains two programs.

- The Team Server accepts check-ins. It stores the results. It gives the
  operator a console.
- The Beacon calls the Team Server. It runs the commands. It returns the output.

The repository publishes one Beacon file. That file has the shape of a Beacon.
It does not run any command. See `cmd/beacon/main.go`.

The working Beacon is in `lab/beacon-impl`. A lab operator builds it with
`make lab-beacon`.

## The packages

| Package | Purpose |
|---|---|
| `internal/c2proto` | The data that both programs send. It holds one definition of each type. |
| `internal/agentstore` | The agent table and the task queue of each agent. |
| `internal/transport` | Authentication, TLS material, and the HTTP limits. |
| `internal/opslinux` | The Linux `prctl` call. It changes the process name. |
| `cmd/team-server` | The operator program. |
| `cmd/beacon` | The published Beacon interface. |
| `lab/beacon-impl` | The working Beacon. It runs the commands. |

## The sequence of one check-in

The Beacon repeats these steps.

1. The Beacon waits for the jitter interval.
2. The Beacon sends a `GET` request to `/checkin`.
3. The Beacon sends the agent ID and the host name in headers.
4. The Beacon sends the shared secret in the `Authorization` header.
5. The Team Server checks the shared secret.
6. The Team Server checks the agent ID.
7. The Team Server registers the agent, or refreshes the last-seen time.
8. The Team Server takes one task from the queue.
9. The Team Server sends the task, or it sends the status `204`.

The Team Server releases its lock before it writes the response. A slow client
does not block the other agents.

## The sequence of one result

1. The Beacon runs the command with `/bin/bash -c`.
2. The Beacon reads the standard output and the standard error.
3. The Beacon sends a `POST` request to `/results`.
4. The Team Server checks the shared secret.
5. The Team Server checks the agent ID and the task ID.
6. The Team Server sends the result to the console.

The Beacon repeats the send up to three times. The Beacon reads the status
code of every send. A status other than `200` starts another attempt.

## The jitter

The Beacon adds a random change to the base interval.

```text
interval = base + random_between(-limit, +limit)
limit    = base * jitter_percent / 100
```

With the default values the result is between 8 and 12 seconds. The README of
the previous version stated "10 seconds plus 20 percent". That described a
range of 10 to 12 seconds. The code has always produced a range of 8 to 12
seconds.

The Beacon uses `math/rand/v2`. The module needs no random seed. The function
`Seed` in `math/rand` does nothing after Go 1.24.

## The process name

The Beacon calls `prctl` with `PR_SET_NAME`. The kernel field holds 16 bytes.
The longest name has 15 characters, because the kernel stores a null
terminator.

The call changes only the `comm` field. The call does not change the command
line, the executable path, or the parent process. A process that has the same
parent as a real kernel thread still differs from that thread in the other
fields. See `docs/DETECTION.md`.

## What each version fixed

| Version | Change |
|---|---|
| The first version | The repository did not compile. It had no authentication. It used plain HTTP. |
| This version | The repository compiles. It uses a shared secret and TLS. It has tests. |
