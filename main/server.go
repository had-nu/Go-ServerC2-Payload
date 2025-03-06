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
}