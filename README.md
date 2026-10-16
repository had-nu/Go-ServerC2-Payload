# Go-ServerC2-Payload

[![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

A C2 teaching kit written in Go. The kit has two programs. The **Team Server**
accepts check-ins and gives an operator a console. The **Beacon** calls the
Team Server, runs commands, and returns the output.

This repository is a case study. It explains how a command and control
architecture works, and how it looks from the defender side.

> **WARNING:** THIS SOFTWARE EXECUTES COMMANDS THAT A REMOTE CLIENT SENDS. RUN
> IT ONLY IN A CLOSED LAB THAT YOU CONTROL. USE ON A SYSTEM THAT YOU DO NOT OWN
> IS ILLEGAL IN MOST COUNTRIES.

## What this repository publishes

| Program | Path | Runs commands? |
|---|---|---|
| Team Server | `cmd/team-server` | It queues commands. An operator sends them. |
| Interface Beacon | `cmd/beacon` | No. It prints text and stops. |

The working Beacon is in `lab/beacon-impl`. The `.gitignore` file excludes the
`lab` directory, so the working Beacon is **not** in the public repository. See
[docs/THREAT_MODEL.md](docs/THREAT_MODEL.md).

## The documentation

| Document | Purpose |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | How the two programs exchange data |
| [docs/DETECTION.md](docs/DETECTION.md) | How to detect the technique |
| [docs/IOC.md](docs/IOC.md) | The values that a defender searches for |
| [docs/GLOSSARY.md](docs/GLOSSARY.md) | One term for one concept |
| [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) | What an attacker can do |

The documents use the writing style of [ASD-STE100][ste] Issue 9. The style
shortens the sentences. It is **not** a certification against the standard. Read
[docs/WRITING-STYLE.md](docs/WRITING-STYLE.md) before you add a document.

[ste]: https://www.asd-ste100.org/

## Prerequisites

- Go 1.24 or later. Run `go version` to check.
- Linux. The Beacon calls a Linux system call.

## Build

```bash
make build
```

The command writes two files:

- `bin/team-server`
- `bin/beacon-stub`

## Run the Team Server

```bash
make run-server
```

The command listens on `127.0.0.1:8443`. It uses TLS. On the first start it
writes two files in the `.state` directory:

- `.state/c2.key` — the shared secret, mode `0600`
- `.state/team-server.crt` — the certificate, mode `0600`

The program `cmd/team-server` uses `~/.local/share/go-serverc2-payload` for the
state directory when it runs without `make`. The `make` target uses `.state`
in the working directory.

Read the secret with:

```bash
make show-secret
```

## Use the console

The console gives these commands:

```text
c2> agents                 list the registered agents
c2> interact <agent-id>    send the next line to one agent
c2> back                   stop sending to the agent
c2> exit                   stop the team server
```

An example session:

```text
c2> agents
--- agents ---
[ALIVE] id=e51ec4c6 host=ubuntu-target ip=192.0.2.10 last_seen=1s pending=0

c2> interact e51ec4c6
sending to e51ec4c6
e51ec4c6 c2> id
task 1792179166429158512 queued for e51ec4c6

[*] result from e51ec4c6 (task 1792179166429158512):
uid=1000(operator) gid=1000(operator)
```

## Build the working Beacon

**CAUTION:** THE WORKING BEACON RUNS REMOTE COMMANDS. BUILD IT ONLY FOR A LAB
THAT YOU CONTROL.

```bash
make lab-beacon
```

The command writes `bin/beacon`. The file has no symbol table. The build uses
the flag `-ldflags="-s -w"`.

Run it against a Team Server on the same machine:

```bash
./bin/beacon \
  -server https://127.0.0.1:8443 \
  -secret-file .state/c2.key \
  -ca .state/team-server.crt \
  -sleep 10s \
  -jitter 20
```

The flag `-ca` makes the Beacon check the Team Server certificate. Without this
flag the Beacon accepts any certificate. That is acceptable in a closed lab
network. It is not acceptable anywhere else.

The flag `-sleep` sets the base interval. The flag `-jitter` sets the largest
random change, in percent. With the values above the interval is between 8 and
12 seconds.

## Test

```bash
make check
```

The command runs the format check, `go vet`, the tests with the race detector,
and the writing-style check.

The coverage of the packages:

| Package | Coverage |
|---|---|
| `internal/c2proto` | 100% |
| `internal/agentstore` | 91% |
| `internal/transport` | 82% |
| `internal/opslinux` | 83% |

## Check the writing style

The repository uses the writing style of ASD-STE100 Issue 9. The tool
`tools/stelint` checks the rules that a program can decide.

```bash
make ste
```

The tool checks these rules:

| Rule | Check |
|---|---|
| 2.1 | A multi-word noun has 3 words at most. |
| 4.2 | The text has no contraction. |
| 5.1 | An instruction has 20 words at most. |
| 6.3 | A sentence has 25 words at most. |
| 6.6 | A paragraph has 6 sentences at most. |
| 8.1 | The text has no semicolon. |
| 1.1 | The text has no modal that the dictionary does not approve. |
| 1.2 | The text does not use `CHECK` or `TEST` as a verb. |
| 1.3 | The text does not use `ABOUT` for a value that is not exact. |

The tool cannot check the approved meaning of a noun, or the approved part of
speech of a technical noun. A writer checks those by hand.

The tool does not certify a document against the standard. A writer claims only
that the document follows the writing style. See
[docs/WRITING-STYLE.md](docs/WRITING-STYLE.md).

When a document names a form that the dictionary does not approve, add a
directive to that line:

```markdown
| `CHECK (n)` with `do a check of` | `CHECK (v)` <!-- stelint:allow 1.2 --> |
```

## The changes in this version

The previous version did not compile. It had 8 redeclaration errors. Its
README described paths that did not exist.

This version changes the following:

- The repository compiles. It uses `cmd/` and `internal/` packages.
- Both handlers check a shared secret.
- The Team Server uses TLS by default.
- The Team Server sets the HTTP limits and a graceful shutdown.
- The agent ID has a length check. The console no longer panics.
- The Beacon checks the status code of every send.
- The Beacon no longer calls `math/rand.Seed`, which does nothing after Go 1.24.
- The Beacon no longer copies a name over `os.Args`, which never worked.
- The code has tests, a `Makefile`, a linter configuration, and CI.
- The documentation describes the detection side.

See [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) for the full list.

## Legal

Use this software only on systems that you own or that you have written
permission to test. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).

[docs/WRITING-STYLE.md]: docs/WRITING-STYLE.md
