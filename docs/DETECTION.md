# Detection

This document describes what this tool looks like from the outside. Use it to
write detections.

The text follows the writing style of ASD-STE100 Issue 9. The style does not
certify the text against the standard.

## Host indicators

This section states the values that the author measured on a Linux host. Each
statement names the file that gives the value.

### What the mask changes, and what it does not

The Beacon writes the string `[kworker/u4:2]` to the `comm` field with the
system call `prctl`. The kernel stores the value in the `comm` field only.

The author measured the result. The table shows the value of each file.

| Source | Value after the mask | Hides the process? |
|---|---|---|
| `/proc/<pid>/comm` | `[kworker/u4:2]` | Yes |
| `ps -o comm=` | `[kworker/u4:2]` | Yes |
| `/proc/<pid>/cmdline` | the real command line | **No** |
| `ps -o args=` | the real command line | **No** |
| `/proc/<pid>/exe` | the real path to the binary | **No** |

A rule must read a field that the mask does not change. The fields `cmdline`
and `exe` keep the truth.

### The process name

Real kernel worker threads have different names. The author measured these
names on a Linux host:

```text
kworker/R-rcu_gp
kworker/0:0H-kblockd
kworker/4:1-i915-unordered
pool_workqueue_release
```

A kernel thread name has no opening bracket. A masked userspace process usually
has one.

A weak rule searches for a `comm` value that begins with `[` and ends with `]`.
Many programs write a bracketed name. Do not build a rule on the bracket alone.

### The executable file

The author measured this value on a Linux host.

| Process | `/proc/<pid>/exe` |
|---|---|
| A shell | A readable link to `/usr/bin/bash` |
| A kernel worker thread | No entry |
| `kthreadd`, which is process 2 | A link that a normal user cannot read |

A kernel worker thread has no entry at the path `/proc/<pid>/exe`. The
exception is process 2. A rule must not treat process 2 as a userspace process.

This check is strong. Combine it with the `comm` value.

### The parent process

Real kernel threads have a parent process ID of `2`. The author measured this
value. A userspace process cannot have a parent of `2`.

This is the strongest single check for this technique:

```text
A process has comm = "[kworker/u4:2]" AND ppid = 2
```

That combination does not happen. Write the rule with this logic.

### The parent of the parent

The author measured the ancestry of a kernel worker thread:

```text
pid 90167  ppid 2      comm kworker/4:1-i915-unordered
pid 2      ppid 0      comm kthreadd
```

The chain ends at process ID `0`. The `proc` file system has **no** entry for
process ID `0`. A rule cannot read `/proc/0/stat`, and it cannot read the name
`swapper/0` from `proc`.

A rule must stop the walk at process ID `0` and test for that value. It must
not try to read a name at process ID `0`.

### The field order in `/proc/<pid>/stat`

The second field of `/proc/<pid>/stat` is the `comm` value **in parentheses**.
The fourth field is the parent process ID.

A shell that reads these fields with `awk '{print $4}'` gives a wrong answer
when the `comm` value contains a space. The shell splits the line on the space
before it reads the field. The mask `[kworker/u4:2]` has no space, so the simple
command gives the right answer for this technique. A rule for other names needs
a parser. The parser reads the last `)` before it reads the fields that follow.

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
