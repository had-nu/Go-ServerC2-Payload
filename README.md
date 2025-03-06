# Go C2 Server and Payload

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://opensource.org/licenses/Apache-2.0)

A simple Command and Control (C2) server and payload written in Go. The server listens for incoming TCP connections on port 8080 and executes commands received from the payload client, returning the output. This project is designed for educational purposes and to demonstrate basic network programming in Go.

## Description

This repository contains two main components:
- **`server.go`**: A TCP server that listens on port 8080, accepts commands from clients, and executes them using the system's shell (`/bin/sh` on Linux). It supports multiple commands per connection.
- **`payload.go`**: A client that connects to the server, sends a predefined list of commands, and displays the responses.

Both components use `bufio.Reader` for robust handling of variable-length input/output and are configured for Linux environments.

---

## Ethical Hacking Disclaimer

This project is intended **solely for educational purposes and ethical security research**. It demonstrates basic C2 functionality and should only be used in controlled environments where you have explicit permission to test (e.g., your own systems or networks). Unauthorized use of this code to harm systems, networks, or individuals is strictly prohibited and illegal. It’s open for collaboration, but use it responsibly in authorised settings only. My intent is to empower security pros and students, not enable misuse.

**Use responsibly and ethically.**

---

## Prerequisites

- Go 1.24 or later installed (`go version` to check).
- A Linux environment (tested on Ubuntu/Debian).
- Basic networking setup (ensure port 8080 is open and accessible).

## Usage

1. Clone the repository:
   ```bash
   git clone https://github.com/had-nu/Go-ServerC2-Payload.git
   cd Go-ServerC2-Payload
   ```
2. Start the server
``` bash
go run server.go
```

3. Run the payload
``` bash
go run payload.go
```

The payload will connect to the server (`default: 127.0.0.1:8080`), send commands, and display the responses with a 2-second delay between each command.

The server listens on :8080. Modify the port in `server.go` if needed:
``` bash
ln, err := net.Listen("tcp", ":8080")
```
The payload connects to `127.0.0.1:8080`. You can update the IP/Port in `payload.go` as well:
``` bash
conn, err := net.Dial("tcp", "127.0.0.1:8080")
```
Edit the `commands` slice in `payload.go` to customaise the commands sent:
``` bash
commands := []string{
    "whoami",
    "ip addr",
    "pwd",
    "ls",
}
```

The Server Output must be something like that:
``` bash
$ go run server.go
C2 server started on port 8080
Listening on [::]:8080
New connection from 127.0.0.1:54321
Received command: /usr/bin/whoami
Response sent: user
Received command: /sbin/ip addr
Response sent: 1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
[... truncated ...]
```

---
### License

Licensed under the Apache License 2.0. See LICENSE for details.
