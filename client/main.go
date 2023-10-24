package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	serverAddress := "localhost:9090"

	// Connecta ao servidor
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to the FS Tracker server")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a command (e.g., REGISTRATION, UPDATE, LOCATE, or QUIT): ")
		command, _ := reader.ReadString('\n')
		command = command[:len(command)-1] // Remove a quebra de linha

		// Mandar o comando para o servidor
		_, err := conn.Write([]byte(command + "\n"))
		if err != nil {
			fmt.Println("Error sending command to the server:", err)
			break
		}

		// Le e imprime a resposta do servidor
		response, err := bufio.NewReader(conn).ReadString('\n')
		if command == "QUIT" {
			break
		}

		if err != nil {
			fmt.Println("Error receiving response from the server:", err)
			break
		}

		fmt.Println("Server Response:", response)
	}
}
