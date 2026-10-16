# Threat model

This document describes what an attacker can do with this code. It describes
the risk in ordinary language. The risk needs precise words. The writing style
of ASD-STE100 shortens the sentences. It does not shorten the risk.

## What this repository publishes

The repository publishes two programs.

- The Team Server. It needs a terminal and an operator. It runs on the
  administrator machine.
- The Interface Beacon. It runs no command. It prints text and stops.

The repository does not publish a working Beacon. The `.gitignore` file
excludes the `lab` directory. The working Beacon is not in the history of the
public repository either. See "The previous version" below.

## The trust boundary

The Trust boundary is between the Team Server and everything that reaches it
over the network.

```text
operator  ---terminal--->  Team Server  ---TLS--->  Beacon
                trust boundary 1        trust boundary 2
```

Trust boundary 1 separates the operator from the server process. Trust
boundary 2 separates the server from every network client.

## The threats

### T1. An attacker registers a fake Beacon

**Before the fix.** The first version had no authentication. Any host that
reached port 8080 sent a `GET /checkin` request with an `X-Agent-ID` header.
The Team Server registered the host as an agent. The attacker read the tasks
that the operator queued for other agents. The attacker posted false results.

**Now.** Both handlers check the shared secret. The Team Server compares the
value in constant time. A request without the value receives `401`.

**Residual risk.** The shared secret is a single value. Anyone who reads
`c2.key` has the ability to register a Beacon and to read results. The file has
the permission `0600`. There is no second factor and no per-agent secret.

### T2. An attacker reads the traffic

**Before the fix.** The traffic was plain HTTP on port 8080. Anyone on the path
read the commands and the output. The output of a `cat /etc/shadow` attempt is
useful to an attacker.

**Now.** The Team Server uses TLS by default. The Beacon accepts the
self-signed certificate.

**Residual risk.** A self-signed certificate does not prove the identity of the
server. An attacker in the middle can present another certificate. The Beacon
does not check the server by default. An operator must pass the certificate with
the flag `-ca`. The documentation states this.

### T3. An attacker crashes the Team Server

**Before the fix.** The console used the expression `ID[:8]`. A Beacon sent the
header `X-Agent-ID: ab`. The operator typed the command `interact ab`. The
server stopped with a panic. Any host on the network can trigger this. The
server had no authentication, so the trigger was simple.

**Now.** The Team Server checks the agent ID before it stores the ID. The check
requires at least 8 characters and at most 64. The console uses a function that
returns the whole value when the value is short. The test
`TestEndToEndShortAgentIDDoesNotPanic` covers this.

### T4. An attacker fills the memory of the Team Server

**Before the fix.** The handler for results called `io.ReadAll` on the request
body. No limit. A request with a large body used the memory of the server.

**Now.** The handler wraps the body in `http.MaxBytesReader` with a limit of 1
MiB. The server also limits the header size and sets the timeouts. The store
refuses more than 64 tasks for one agent.

### T5. An attacker writes false lines in the server log

**Before the fix.** The handler wrote the value of the `X-Agent-ID` header and
the value of the `X-Hostname` header in the log. A client sent a header with a
newline. The client wrote its own lines in the log. A client sent an escape
sequence. The client changed the appearance of the log.

**Now.** The Team Server removes the control characters from these headers. It
truncates the value to 255 characters. The test
`TestEndToEndLongHostnameIsTruncated` covers the length.

### T6. An attacker uses the tool against a third party

**The risk is unchanged by the code.** A C2 runs the commands that an operator
sends. An attacker with the Team Server can use it on any host that the
attacker can reach. The code does not restrict the target.

The mitigation is outside the code:

- the operator needs written permission for every target
- the Team Server binds to the loopback interface by default in the
  documentation
- the shared secret prevents an unknown host from registering

### T7. An attacker finds the shared secret on disk

An analyst who reads `c2.key` can register a Beacon. The analyst can read the
results of every agent. See T1.

## The accepted risks

| Risk | Why it is accepted |
|---|---|
| One shared secret for all agents | A lab has few agents. A second factor has no value here. |
| A self-signed certificate | A lab has no certificate authority. |
| The Beacon runs any command | That is the purpose of a C2. |
| `math/rand` for the jitter | The jitter does not need a cryptographic source. |
| The Beacon does not verify the server | The lab network is closed. An operator passes `-ca` when the network is not closed. |

## What this version does not do

The Beacon does not use these techniques:

- it does not change the executable path
- it does not change the `argv` array in memory
- it does not inject code into another process
- it does not persist across a restart
- it does not remove itself

## The previous version

The first version of the project did not compile. The file `payload.go` was in
the `.gitignore` file. The repository published the stub file only. The stub
file has empty function bodies. The stub program stops with the status `0`.

That version was safe by accident. It was not a design. A reader of the
repository cannot run a Beacon. This version keeps that property on purpose.
The published Interface Beacon still runs nothing. The difference is that the
repository now compiles, has tests, and states the design in writing.
