# Indicators of compromise

This document lists the values that a defender can search for.

The text follows the writing style of ASD-STE100 Issue 9. The style does not
certify the text against the standard.

## Host values

| Type | Value | Where to look | Strength |
|---|---|---|---|
| Process name | `[kworker/u4:2]` | `/proc/<pid>/comm`, `ps -eo comm` | Weak alone |
| Process name prefix | `[kworker` | `/proc/<pid>/comm` | Weak alone |
| Parent process ID | `2` | `/proc/<pid>/stat` field 4 | Strong with a masked name |
| Missing executable | no file | `/proc/<pid>/exe` | Strong with a masked name |
| Ancestry | `swapper/0` with process ID `0` | `/proc/<pid>/stat` | Strong with a masked name |

The default masked name has 14 characters. The kernel allows 15.

## Network values

| Type | Value | Where to look | Strength |
|---|---|---|---|
| Header name | `X-Agent-ID` | HTTP request | Strong |
| Header name | `X-Hostname` | HTTP request | Medium |
| Path | `/checkin` | HTTP request | Weak alone |
| Path | `/results` | HTTP request | Weak alone |
| Method | `GET` on `/checkin` | HTTP request | Weak alone |
| Method | `POST` on `/results` | HTTP request | Weak alone |
| Body key | `"task_id"` | HTTP request or response | Medium |
| Body key | `"output"` | HTTP response | Medium |
| Certificate common name | `Go-ServerC2-Payload Team Server` | TLS certificate | Strong |
| Certificate organization | `Educational lab use` | TLS certificate | Strong |
| Auth scheme | `Bearer` with 64 hex characters | HTTP request | Medium |

## File system values

| Type | Value | Where to look |
|---|---|---|
| File name | `c2.key` | the state directory |
| File name | `team-server.crt` | the state directory |
| File name | `team-server.key` | the state directory |
| Directory name | `go-serverc2-payload` | `$HOME/.local/share` |

The state directory holds the shared secret. An analyst who finds `c2.key` has
the secret. The analyst can register a new Beacon or read the results of an
existing Beacon.

Treat a `c2.key` file as a high-value indicator. Alert on it.

## Strings in the binary

The lab Beacon contains these strings. A defender can search a binary for them.

| String | Reason |
|---|---|
| `/checkin` | the path |
| `/results` | the path |
| `X-Agent-ID` | the header |
| `X-Hostname` | the header |
| `[kworker/u4:2]` | the masked name |
| `/bin/bash` | the interpreter |

The published Interface Beacon also contains these strings. That program runs
nothing. A search for the strings gives a false positive on the Interface
Beacon.

## What a defender cannot see

This tool does not use these techniques:

- it does not change the executable path
- it does not change the `argv` array in memory
- it does not inject code into another process
- it does not persist across a restart
- it does not remove its own files

A defender does not need to search for those techniques.

## A note on the previous version

The first version sent all traffic over plain HTTP on port 8080. It sent no
shared secret. A defender can search an old capture for a request to
`/checkin` with the header `X-Agent-ID` and no `Authorization` header.
