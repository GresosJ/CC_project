package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func handleClient(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Client connected:", conn.RemoteAddr())

	// Create a bufio reader to read messages from the client
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Client disconnected:", conn.RemoteAddr())
			return
		}
		message = strings.TrimSpace(message)
		fmt.Println("Received message from client:", message)

		if message == "ok" {
			fmt.Println("Client sent 'ok'. Closing the connection.")
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		go handleClient(conn)
	}
}
