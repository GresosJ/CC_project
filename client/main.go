package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// global vars
var estimatedRTT time.Duration
var devRTT time.Duration

var brokenFiles map[string][][]byte

func main() {
	serverAddress := "10.0.2.10:9090"
	heartbitInterval := 5 * time.Second

	brokenFiles = make(map[string][][]byte)

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
		fmt.Printf("Erro ao tentar criar um listener UDP: %v\n", err)
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

			_, err := conn.Write([]byte("LOCATE " + fileID + "\n"))
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

			// Verifique se há pelo menos um endereço IP disponível
			if len(nodesInfos) > 0 {
				for _, nodeIP := range nodesInfos {
					firstNodeIP = nodeIP
					break
				}
			} else {
				fmt.Println("Nenhum endereço IP disponível.")
			}

			// Abrir "Conexao" UDP
			udpNodeConn, err := openUDPConn(firstNodeIP + ":8081")
			if err != nil {
				fmt.Println("Erro ao abrir a conexão UDP", err)
				return
			}
			defer udpNodeConn.Close()

			transferedFile, err := transferAndAssembleFile(udpNodeConn, fileID)
			if err != nil {
				fmt.Println("Erro ao tentar juntar o ficheiro", err)
				return
			}

			// Guarda o ficheiro na pasta files
			err = os.WriteFile(getPath(fileID), transferedFile, 0644)
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
	<-done
	udpListener.Close()
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

	// Use Stat para obter informações sobre o arquivo
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Imprime o tamanho do arquivo
	fmt.Printf("Tamanho do arquivo: %d bytes\n", fileInfo.Size())


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
			ipAddress := strings.Split(parts[2], ":")[0] // Combine partes do IP, se necessário
			nodeInfoMap[nodeName] = ipAddress
		}
	}

	return nodeInfoMap, nil
}

func transferAndAssembleFile(conn *net.UDPConn, fileID string) ([]byte, error) {

	// map para guardar os blocos recebidos
	dataInBlocks := make(map[int][]byte)
	blockID := 0

	for {
		// Solicita bloco
		requestDataBlock(conn, fmt.Sprintf("%d", blockID), fileID)

		// Aguarda dados do bloco
		buffer := make([]byte, totalBlockSize)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Erro ao receber dados do bloco", err)
			return nil, err
		}

		// Verifica a integridade do bloco recebido
		dataInBlocks[blockID] = checkReceivedDataBlock(buffer[:n])
		if err != nil {
			fmt.Println("Erro ao verificar a integridade do bloco", err)
			return nil, err
		} else if dataInBlocks[blockID] != nil {

			// Confirma o bloco
			confirmData(conn, fmt.Sprintf("%d", blockID), fileID)
			fmt.Println("DataBlock recebido com sucesso...")
	
			if(string(dataInBlocks[blockID]) == "END_OF_FILE") {
				break
			}
	
			blockID++
		}

	}

	// Junta o arquivo
	var assembledFile []byte
	for i := 0; i < blockID; i++ {
		assembledFile = append(assembledFile, dataInBlocks[i]...)
	}

	return assembledFile, nil
}

// Funcao onde vai estar toda a logistica dos requests
func handleIncommingRequests(conn *net.UDPConn, done chan struct{}) {

	defer close(done)

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

func handleUDPRequest(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	request := string(data)
	parts := strings.Fields(request)

	if len(parts) < 3 || parts[0] != "REQUEST" {
		fmt.Println("Formato de REQUEST inválido", request)
		return
	}

	blockID := parts[1]
	fileID := parts[2]

	dataBlocks, ok := brokenFiles[fileID]
	if !ok {
		blocks, err := breakFileInBlocks(getPath(fileID))
		if err != nil {
			fmt.Println("Erro ao dividir o arquivo em blocos:", err)

		}

		brokenFiles[fileID] = blocks
		dataBlocks = blocks
	}

	dataBlock, isLastBlock, err := getDataBlock(fileID, blockID, dataBlocks)
	if err != nil {
		fmt.Println("Erro ao obter datablock ", err)
		return
	}

	// Cria um channel para o Timout
	timeout := time.After(TimeoutDuration())

	// Tempo de início para calcular o SampleRTT
	start := time.Now()

	// Enviar o datablock
	sendDataBlock(conn, addr, blockID, fileID, dataBlock)

	select {
	case <-timeout:
		fmt.Println("Tempo limite atingido. Reenviando a solicitação...")
		handleUDPRequest(conn, addr, data)
		return
	default:
		_, _, err := conn.ReadFromUDP(data)
		if err != nil {
			fmt.Println("Erro ao esperar pela confirmação", err)
			return
		}
	}

	if isLastBlock {
		fmt.Println("Fim do ficheiro")
		message := "END_OF_FILE"
		sendDataBlock(conn,addr,blockID,fileID,[]byte(message))
		return
	}

	// Calcula SampleRTT
	SampleRTT := time.Since(start)

	// Atualiza o EstimatedRTT e DevRTT
	updateRTTParameters(SampleRTT)

}

func TimeoutDuration() time.Duration {
	return estimatedRTT + 4*devRTT
}

func updateRTTParameters(SampleRTT time.Duration) {
	alpha := 0.125
	beta := 0.25

	estimatedRTT = time.Duration((1-alpha)*float64(estimatedRTT) + alpha*float64(SampleRTT))
	devRTT = time.Duration((1-beta)*float64(devRTT) + beta*float64(time.Duration(math.Abs(float64(SampleRTT-estimatedRTT)))))

	// (Opcional) Imprimir estimatedRTT e devRTT para fins de depuração
	fmt.Printf("estimatedRTT: %v, devRTT: %v\n", estimatedRTT, devRTT)
}

func getDataBlock(fileID, blockID string, blocks [][]byte) ([]byte, bool, error) {
	index, err := strconv.Atoi(blockID)
	if err != nil {
		fmt.Println("Erro ao converter blockID para inteiro:", err)
		return nil, false, err
	}

	if index < 0 || index >= len(blocks) {
		fmt.Println("Indice do bloco fora do intervalo:", index)
		return nil, false, fmt.Errorf("bloco de id inválido")
	}

	isLastBlock := index == len(blocks)-1

	return blocks[index], isLastBlock, nil
}

func getPath(fileID string) string {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Erro ao obter o caminho do executável:", err)
		return ""
	}
	projectDir := filepath.Dir(executablePath)
	filePath := filepath.Join(projectDir, "..", "files", fileID)
	if err != nil {
		fmt.Println("Erro ao dividir o arquivo em blocos:", err)
		return ""
	}

	return filePath
}

