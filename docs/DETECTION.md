# Detection

This document describes what this tool looks like from the outside. Use it to
write detections.

The text follows the writing style of ASD-STE100 Issue 9. The style does not
certify the text against the standard.

## Host indicators

### The process name

The Beacon writes the string `[kworker/u4:2]` to the `comm` field.

Real kernel worker threads have different names. A typical kworker name has
this form:

```text
kworker/0:1-events
kworker/u16:2
```

The mask in this project ends with a colon and a digit, and it has no process
number before the colon. A rule can search for a `comm` value that begins with
`[kworker` and ends with `]`.

A strong rule must also check the other fields. See "Why the name alone is weak".

A weak signal is a name that begins with `[`. Many programs write a bracketed
name. Do not build a rule on the bracket alone.

### The executable

The Beacon is a single static binary. It has no interpreter in the process
tree. A shell calls it, so the parent process is often a shell that has already
exited.

Check these fields together:

- the `comm` value
- the name of the executable in `/proc/<pid>/exe`
- the name of the executable in `/proc/<pid>/cmdline`
- the first start of the process in `/proc/<pid>/stat`

A real kernel thread has no executable file. The path `/proc/<pid>/exe` does not
exist for a real kernel thread. This is a strong check.

### The parent process

Real kernel threads have a parent of `2`, which is the `kthreadd` process. A
userspace process cannot have a parent of `2`.

This is the strongest single check for this technique:

```text
A process has comm = "[kworker/u4:2]" AND ppid = 2
```

That combination does not happen. Write the rule with this logic.

### The parent of the parent

A real kernel thread has an ancestry of `kthreadd` and then `swapper/0` with a
process ID of `0`. Check the second parent for that value.

## Network indicators

### The request pattern

The Beacon calls `/checkin` with the `GET` method. The Beacon calls `/results`
with the `POST` method. The Beacon uses no other path.

The request has these headers:

| Header | Value |
|---|---|
| `X-Agent-ID` | A UUID-like value with a dash between the groups |
| `X-Hostname` | The host name of the machine |
| `Authorization` | `Bearer` and 64 hexadecimal characters |

A rule can search for a request with an `X-Agent-ID` header. That header name
is unusual. Few legitimate programs send it.

### The interval

The Beacon waits between 8 and 12 seconds by default. The Beacon repeats the
interval with a random change.

A fixed interval is easy to detect. A random interval is harder. A statistical
rule can measure the gap between requests to one path and one client. A period
from 8 to 12 seconds with low variance inside each gap, and high variance
between gaps, does not match a normal browsing pattern.

### The size

A check-in request with no task has an empty body. A request with a task has a
small JSON body.

```json
{"task_id":"1792179166429158512","command":"id"}
```

A result has a JSON body with three fields: `task_id`, `output`, and `error`.
The `output` field often contains a shell transcript.

### The TLS certificate

The Team Server generates a self-signed certificate on the first start. The
certificate has these properties:

| Property | Value |
|---|---|
| Subject common name | `Go-ServerC2-Payload Team Server` |
| Subject organization | `Educational lab use` |
| Validity | 365 days |
| Extended key usage | Server authentication |

A rule can match this common name. A self-signed certificate that presents this
name is an indicator.

The certificate does not change the TLS fingerprint between runs. The operator
keeps the same key file. The `Makefile` target `run-server` stores the state in
the `.state` directory.

## Why the name alone is weak

An analyst who knows the technique can defeat the process name alone. The
analyst reads the `comm` field and sets it to any value.

Therefore a detection must combine at least two checks:

1. the `comm` value
2. the absence of `/proc/<pid>/exe`
3. the parent process ID
4. the executable in `/proc/<pid>/cmdline`

A check on the `comm` value alone gives false positives with software that
writes bracketed names. A check on `ppid = 2` alone gives false positives with a
process that the kernel created.

## What this tool does not do

The Beacon in this repository does not use these techniques:

- it does not change the executable path
- it does not change the `argv` array in memory
- it does not inject into another process
- it does not persist across a restart
- it does not remove itself

The first version of the project had a loop that copied the name over
`os.Args`. That loop did nothing. Each conversion of a string to a byte slice
made a new slice. The write went into a temporary. `git log` shows the removal.

An analyst who reads that commit can use it. It tells the analyst that the
author tried to change the command line and did not succeed.
