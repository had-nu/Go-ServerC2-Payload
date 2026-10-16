# Glossary

This document defines one term for each concept in this project. A writer uses
the same term for the same concept in every document.

The definitions follow the structure of ASD-STE100 rule 1.5. Most terms are
technical nouns in category 6, "Systems, components and circuits, their
functions, configurations, and parts". Category 19, "Computer science,
information and communication technology", holds the network terms. Category 3,
"Tools and support equipment, their parts, and locations on them", holds the
process terms.

## The programs

**Team Server** (noun, technical noun, category 6) — The program that accepts
check-ins, stores results, and gives the operator a console. The first version
of the project called this program the "server". Do not use "server" alone.

**Beacon** (noun, technical noun, category 6) — The program that calls the Team
Server, runs commands, and returns output. Do not use "payload", "implant", or
"client" for this program. Those terms mean something else in security writing.

**Lab Beacon** (noun) — The working Beacon in `lab/beacon-impl`. It is not
published.

**Interface Beacon** (noun) — The Beacon file in `cmd/beacon`. It has the shape
of a Beacon and runs nothing.

## The network terms

**Check-in** (noun, technical noun, category 19) — One request from a Beacon to
`/checkin`. Do not use "heartbeat", "ping", or "poll" for this request.

**Result** (noun, technical noun, category 19) — One message that gives the
output of one executed command. Do not use "response", "output", or "reply" for
this message. "Output" is the field inside the message.

**Task** (noun, technical noun, category 19) — One command that the Team Server
sends to one Beacon.

**Task queue** (noun, technical noun, category 6) — The list of tasks that wait
for one agent. The limit is 64 tasks.

**Agent** (noun, technical noun, category 19) — One registered Beacon. The
dictionary of ASD-STE100 also approves the word "agent". Use the same word.

**Shared secret** (noun) — The value that both programs send in the
`Authorization` header. The value has 64 hexadecimal characters. The Team
Server compares the value in constant time.

Do not use "token", "key", or "API key" for this value. "Key" is already a
separate concept in this project. See **TLS key**.

**Masked name** (noun) — The value that the Beacon writes to the `comm` field.
The default value is `[kworker/u4:2]`.

Do not use "spoofed name", "fake name", or "alias" for this value.

## The system terms

**Last-seen time** (noun) — The time of the most recent check-in from one
agent. The console shows the agent as dead when the last-seen time is more
than 30 seconds in the past.

**Time to live** (noun) — The 30 second interval that decides whether the
console shows an agent as alive or dead. Do not use "timeout". The name "timeout"
already describes the HTTP limits.

**Jitter** (noun, technical noun, category 7) — The random change that the
Beacon adds to the base interval. The default value is 20 percent.

**Jitter interval** (noun) — One computed wait between two check-ins. The
default range is 8 to 12 seconds.

Do not use "sleep interval" or "delay". "Sleep" names the Go function that waits.

**Base interval** (noun) — The constant that the Beacon adds to the jitter. The
default value is 10 seconds.

## The host terms

**Process name** (noun, technical noun, category 3) — The value that the `ps`
and `top` commands show. The kernel stores the value in the `comm` field. The
field holds 16 bytes.

Do not use "process title", "window title", or "task name" for this value.

**Kernel thread** (noun, technical noun, category 3) — A thread that the kernel
creates. A kernel thread has a parent process ID of 2. It has no executable
file.

## The rules for this glossary

1. Use one term for one concept. Do not alternate between two terms.
2. Do not use a term that the project defines in another way.
3. Add a new term here before the first use of a new concept in any document.
4. Keep the category number with the term. The category decides where the term
   comes from, under rule 1.5 of the standard.
