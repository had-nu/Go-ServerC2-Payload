package main

import (
	"fmt"
	"net"
	"time"
	"bufio"
)

func main() {
	// C2 server connection
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Error connecting to C2 server: ", err)
		return
	}
	defer conn.Close()

	// Send commands to C2 server
	commands := []string{
		"whoami",
		"ip addr",
		"pwd",
		"ls",
	}

	for _, command := range commands {
		// Send the command to C2 server
		_, err = conn.Write([]byte(command + "\n"))
		if err != nil {
			fmt.Println("Error sending command: ", err)
			return
		}

		// Read the C2 server response
		reader := bufio.NewReader(conn)
        response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading response: ", err)
			return
		}

		// Display the response
		fmt.Println(response)

		// Waiting before send another command
		time.Sleep(2 * time.Second)
	}
}