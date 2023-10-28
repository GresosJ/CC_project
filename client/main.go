package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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

	registration(conn)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a command (UPDATE, LOCATE, or QUIT): ")
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

		fmt.Print(response)
	}
}

func registration(conn net.Conn) {
	filesDir := "files"
	command := "REGISTRATION"
	fileList, err := listFiles(filesDir)
	if err != nil {
		fmt.Println("Erro ao listar os arquivos na pasta 'files':", err)
		return
	}

	// Combine o comando e a lista de arquivos em uma única string
	message := command + " " + strings.Join(fileList, " ")

	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println("Erro ao enviar comando para o servidor:", err)
		return
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Print(response)
}

func listFiles(directory string) ([]string, error) {
	var fileList []string
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileList = append(fileList, file.Name())
	}

	return fileList, nil
}
