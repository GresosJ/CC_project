package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer conn.Close()

	// Create a bufio reader for user input
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a message ('ok' to exit): ")
		message, _ := reader.ReadString('\n')

		// Send the user's message to the server
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error sending message to server:", err)
			return
		}

		if message == "ok\n" {
			fmt.Println("Closing the connection.")
			return
		}
	}
}
