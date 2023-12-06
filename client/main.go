package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	serverAddress := "localhost:9090"
	heartbitInterval := 5 * time.Second

	// Connecta ao servidor
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		return
	}
	defer conn.Close()

	registration(conn)

	ticker := time.NewTicker(heartbitInterval)
	defer ticker.Stop()

	go watchForFileUpdates(conn, "files")

	go sendHeartbits(conn, ticker.C)

	//Listener para requests UDP

    udpListenerAddr, err := net.ResolveUDPAddr("udp", ":8081")
    if err != nil {
        fmt.Println("Erro ao tentar criar um listener UDP", err)
        return
    }

    udpListener, err := net.ListenUDP("udp", udpListenerAddr)
    if err != nil {
        fmt.Println("Error listening on UDP:", err)
        return
    }
    defer udpListener.Close()

	done := make(chan struct{})

	go handleIncommingRequests(udpListener, done)

	// Inputs
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a command (REQUEST or QUIT): ")
		command, _ := reader.ReadString('\n')
		command = command[:len(command)-1] // Remove a quebra de linha

		switch command {
		case "REQUEST":

			fmt.Print("Enter the file ID to request: ")
			fileID, _ := reader.ReadString('\n')
			fileID = fileID[:len(fileID)-1] // Remove a quebra de linha

			_, err := conn.Write([]byte(command + "\n"))
			if err != nil {
				fmt.Println("Error sending command to the server:", err)
				break
			}

			response, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Error receiving response from the server:", err)
				break
			}

			// Recebe os nodes e os respetivos ip com os files
			nodesInfos, err := parseLocateSuccessMessage(response)
			if err != nil {
				fmt.Println("Error receiving nodes location", err)
				break
			}

			var firstNodeIP string

			// Escolhe o primeiro
			for nodeIP := range nodesInfos {
				firstNodeIP = nodeIP
				break
			}

			// Abrir "Conexao" UDP
			conn, err := openUDPConn(firstNodeIP)
			if err != nil {
				fmt.Println("Erro ao abrir a conexão UDP", err)
				return
			}
			defer conn.Close()

			// Enviar um pedido do numero de blocos que o ficheiro tem
			numBlocks, err := requestNumBlocksUDP(conn, fileID)
			if err != nil {
				fmt.Println("Erro ao receber o numero de blocos do ficheiro", err)
				return
			}

			// pedir os blocos ate que o ficheiro esteja completo
			transferedFile, err := transferAndAssembleFile(conn, fileID, numBlocks)
			if err != nil {
				fmt.Println("Erro ao tentar juntar o ficheiro", err)
				return
			}

			// Guarda o ficheiro na pasta files
			err = os.WriteFile(fileID, transferedFile, 0644)
			if err != nil {
				fmt.Println("Error saving the assembled file:", err)
				return
			}

			return

		case "QUIT":
			// Mandar o comando para o servidor
			_, err := conn.Write([]byte(command + "\n"))
			if err != nil {
				fmt.Println("Error sending command to the server:", err)
			}

			return

		default:
			fmt.Println("Invalid command! Try REQUEST or QUIT.")
		}
	}
}

////////////////// Functions //////////////////

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

func sendHeartbits(conn net.Conn, ticker <-chan time.Time) {
	for range ticker {
		heartbitMessage := "HEARTBIT" + "\n"
		_, err := conn.Write([]byte(heartbitMessage))
		if err != nil {
			fmt.Println("Erro ao enviar comando para o servidor:", err)
			return
		}
		response, err := bufio.NewReader(conn).ReadString('\n')
		if response != "HEARTBIT_ACK\n" {
			conn.Close()
		}
	}
}

// Função para dividir o arquivo
func breakFileInBlocks(filePath string) ([][]byte, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blocks [][]byte
	buffer := make([]byte, maxBlockSize)

	for {
		// Lê o próximo bloco do arquivo
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// Cria uma cópia dos dados lidos para evitar problemas de referência
		block := make([]byte, bytesRead)
		copy(block, buffer[:bytesRead])

		// Adiciona o bloco à lista de blocos
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Pega nos IP que contem o ficheiro
func parseLocateSuccessMessage(message string) (map[string]string, error) {

	nodeInfoMap := make(map[string]string)
	lines := strings.Split(message, "\n")

	for _, line := range lines {

		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "LOCATE_SUCCESS" {

			nodeName := parts[1]
			ipAddress := parts[2]

			nodeInfoMap[nodeName] = ipAddress
		}
	}

	return nodeInfoMap, nil
}

// Request Num of blocks
func requestNumBlocksUDP(conn *net.UDPConn, fileID string) (int, error) {
	requestMsg := fmt.Sprintf("REQUEST_NUM_BLOCKS %s\n", fileID)
	_, err := conn.Write([]byte(requestMsg))
	if err != nil {
		return 0, err
	}

	// Wait for the response
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, err
	}

	numBlocks, err := strconv.Atoi(string(buffer[:n]))
	if err != nil {
		return 0, err
	}

	return numBlocks, nil
}

func transferAndAssembleFile(conn *net.UDPConn, fileID string, numBlocks int) ([]byte, error) {
	const maxBlockSize = 1472

	// map para guardar os blocos recebidos
	receivedBlocks := make(map[int][]byte)

	for blockID := 0; blockID < numBlocks; blockID++ {
		// Solicita bloco
		requestDataBlock(conn, fmt.Sprintf("%d", blockID), fileID)

		// Aguarda dados do bloco
		buffer := make([]byte, maxBlockSize)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Erro ao receber dados do bloco", err)
			return nil, err
		}

		//Confirma o bloco
		confirmData(conn, fmt.Sprintf("%d", blockID), fileID)

		// Verifica a integridade SSSSdo bloco recebido
		checkReceivedDataBlock(buffer[:n])
		if err != nil {
			fmt.Println("Erro ao verificar a integridade do bloco", err)
			return nil, err
		}

		// Guarda os blocos recebidos
		receivedBlocks[blockID] = buffer[:n]

		// ve se e o ultimo bloco
		if blockID == numBlocks-1 {
			// Junta o ficheiro
			var assembledFile []byte
			for i := 0; i < numBlocks; i++ {
				assembledFile = append(assembledFile, receivedBlocks[i]...)
			}
			return assembledFile, nil
		}
	}

	return nil, nil // This should not be reached
}

// Funcao onde vai estar toda a logistica dos requests
func handleIncommingRequests(conn *net.UDPConn, done chan struct{}){

	for {
        select {
        case <-done:
            fmt.Println("Terminando a rotina handleIncommingRequests...")
            return
        default:
            buffer := make([]byte, maxBlockSize)
            n, addr, err := conn.ReadFromUDP(buffer)
            if err != nil {
                fmt.Println("Erro ao ler de UDP", err)
                continue
            }

            handleUDPRequest(conn, addr, buffer[:n])
        }
    }
}

func handleUDPRequest(conn *net.UDPConn, addr *net.UDPAddr, data []byte){
	 
	 request := string(data)

	 parts := strings.Fields(request)
 
	 if len(parts) < 2 {
		 fmt.Println("Formato de REQUEST invalido ", request)
		 return
	 }
 
	 switch parts[0] {
	 case "REQUEST_NUM_BLOCKS":
		 fileID := parts[1]
		 numBlocks, err := getNumBlocksForFile(fileID)
		 if err != nil {
			 fmt.Println("Erro ao obter numero de blocos", err)
			 return
		 }
		 response := fmt.Sprintf("%d\n", numBlocks)
		 _, err = conn.WriteToUDP([]byte(response), addr)
		 if err != nil {
			 fmt.Println("Error ao responder REQUEST_NUM_BLOCKS:", err)
		 }
 
	 case "REQUEST_DATA_BLOCK":
		 fileID := parts[1]
		 blockID := parts[2]
		 dataBlock, err := getDataBlock(fileID, blockID)
		 if err != nil {
			 fmt.Println("Erro ao obter DataBlock", err)
			 return
		 }
 
		 // Send the data block
		 sendDataBlock(conn, blockID, fileID, dataBlock)
 
		// confirmacao 
		 _, _, err = conn.ReadFromUDP(data)
		 if err != nil {
			 fmt.Println("Erro ao esperar pela confirmacao", err)
			 return
		 }
 
	 default:
		 fmt.Println("Tipo de request desconhecido ", parts[0])
	 }
}


func getNumBlocksForFile(fileID string) (int, error){
    blocks, err := breakFileInBlocks(fileID)
    if err != nil {
        return 0, err
    }
    return len(blocks), nil
}

func getDataBlock(fileID, blockID string) ([]byte, error){

    blocks, err := breakFileInBlocks(fileID)
    if err != nil {
        return nil, err
    }

    index, err := strconv.Atoi(blockID)
    if err != nil || index < 0 || index >= len(blocks) {
        return nil, fmt.Errorf("bloco de id invalido")
    }

    return blocks[index], nil
}