# Go Command and Control (C2) Server and Payload

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://opensource.org/licenses/Apache-2.0)

A Command and Control (C2) suite written in Go. This project implements a modern HTTP Beacon architecture designed for educational purposes and to demonstrate Linux-specific evasion techniques. It consists of a central **Team Server** and an HTTP polling **Beacon** payload.

## Description

This repository contains two main components:
- **`main/server/server.go` (Team Server)**: An HTTP REST server that acts as the central hub. It listens on port `:8080` for Beacon check-ins, tracks active agents, and provides an interactive `readline`-based CLI for the operator to issue commands.
- **`main/beacon/payload.go` (Beacon)**: A stealthy Linux client that connects to the Team Server. It periodically polls the server for tasks using an unpredictable Sleep and Jitter cycle, executes commands via `/bin/bash`, and returns the output to the server.

### Key Features
* **HTTP Beaconing:** Communication operates over standard HTTP `GET`/`POST` requests, mimicking normal web traffic rather than dropping raw TCP sockets.
* **Sleep & Jitter:** The beacon does not use static sleep intervals. It calculates a dynamic sleep duration (Base 10s + 20% Jitter) for each check-in, breaking network pattern recognition signatures.
* **Process Spoofing (`prctl`):** Upon execution, the payload aggressively alters its own process name to `[kworker/u4:2]`, allowing it to blend into Linux process lists alongside legitimate kernel worker threads.
* **Operator Console:** The Team Server features an interactive shell to view alive/dead agents and directly drop into sessions to queue tasks.

---

## Ethical Hacking Disclaimer

This project is intended **solely for educational purposes and ethical security research**. It demonstrates C2 functionality and Linux evasion mechanisms. It should only be used in controlled environments where you have explicit permission to test (e.g., your own systems or networks). Unauthorized use of this code to harm systems, networks, or individuals is strictly prohibited and illegal. 

See the Code of Conduct file.

**Use responsibly and ethically.**

---

## Prerequisites

- Go 1.24 or later installed (`go version` to check).
- A Linux environment (tested on Ubuntu/Debian).

## Usage

1. Clone the repository:
	```bash
	git clone https://github.com/had-nu/Go-ServerC2-Payload.git
	cd Go-ServerC2-Payload
	```
2. Start the Team Server:
	```bash
	go run main/server/server.go
	```
	*The server will start on port `8080` and provide a `c2>` prompt.*

3. Compile the Beacon Payload:
	To take full advantage of the evasion techniques, the payload should be built stripping all symbolic and DWARF debugging information.
	```bash
	go build -ldflags="-s -w" -o beacon main/beacon/payload.go
	```

4. Run the Beacon on the target machine:
	```bash
	./beacon &
	```

### Interacting with Agents

Once the beacon is running, it will check-in with the Team Server.

```text
[*] Starting HTTP Team Server on port :8080
c2> 
[+] New Agent Registered: e51ec4c6-a873 (ubuntu-target) from 127.0.0.1
```

Use the `agents` command to verify their status, and `interact <ID>` to issue bash commands:

```text
c2> agents
--- Active Agents ---
[ALIVE] ID: e51ec4c6-a873 | Host: ubuntu-target | IP: 127.0.0.1 | Last Seen: 2s

c2> interact e51ec4c6-a873
e51ec4c6 c2> id
Task 482910 queued for e51ec4c6-a873

[*] Result from e51ec4c6-a873 (Task 482910):
uid=1000(user) gid=1000(user) groups=1000(user),27(sudo)
```

Type `back` to return to the main menu, or `exit` to shut down the Team Server.

---

## Limitations and Future Improvements

While functional for educational purposes, this project has several notable limitations for real-world scenarios:

1. **Platform Dependency**  
   The current payload is rigidly tailored for Linux. Both the command execution (`/bin/bash`) and the process spoofing technique (`SYS_PRCTL`) will fail to compile or execute on Windows or macOS.
2. **Lack of In-Transit Encryption**  
   Communication between the server and payload occurs over plaintext HTTP. While the JSON bodies could be encrypted with AES, implementing full HTTPS via TLS is highly recommended to prevent Deep Packet Inspection (DPI).
3. **No Persistence**
   The beacon only runs in memory while the system is powered on. It does not contain capabilities to survive server reboots (e.g., adding itself to crontab or systemd services).

---
### License

Licensed under the Apache License 2.0. See LICENSE for details.
