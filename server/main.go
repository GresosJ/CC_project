package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// FSNodeInfo representa informações sobre um FS_Node registrado.
type FSNodeInfo struct {
	Address     string
	SharedFiles []string
}

// nodeInfoMap mapeia nomes de FS_Node para informações.
var nodeInfoMap map[string]FSNodeInfo

func main() {
	nodeInfoMap = make(map[string]FSNodeInfo)

	// Porta em que o servidor FS_Tracker escutará
	port := 9090
	if len(os.Args) > 1 {
		portStr := os.Args[1]
		port, _ = strconv.Atoi(portStr)
	}

	// Inicie o servidor FS_Tracker na porta especificada
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("Erro ao iniciar o servidor: %s\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Servidor FS_Tracker ativo na porta %d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Erro na aceitação da conexão: %s\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Use bufio para ler as mensagens do cliente.
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Erro na leitura da mensagem: %s\n", err)
			break
		}
		message = strings.TrimSpace(message)

		// Processar a mensagem com base no protocolo.
		handleMessage(conn, message)
	}
}

func handleMessage(conn net.Conn, message string) {
	parts := strings.Fields(message)

	if len(parts) == 0 {
		fmt.Println("Mensagem vazia")
		sendResponse(conn, " ")
		return
	}

	command := parts[0]

	switch command {
	case "REGISTRATION":
		handleRegistration(conn, parts)
	case "UPDATE":
		handleUpdate(conn, parts)
	case "LOCATE":
		handleLocate(conn, parts)
	case "QUIT":
		handleQuit(conn)
	default:
		sendResponse(conn, "ERROR Comando desconhecido")
	}
}

func handleRegistration(conn net.Conn, parts []string) {
	//Isto esta muito incompleto, mas é só para testar
	nodeName := "node" + conn.RemoteAddr().String()
	ipAddress := conn.RemoteAddr().String()
	sharedFiles := parts[1:]

	//Deveria ser nodeIndomap[nodeName]
	nodeInfoMap[nodeName] = FSNodeInfo{
		Address:     ipAddress,
		SharedFiles: sharedFiles,
	}
	fmt.Println("Registrado FS_Node:", ipAddress)

	sendResponse(conn, fmt.Sprintf("REGISTRATION_SUCCESS %s", nodeName))
}

func handleUpdate(conn net.Conn, parts []string) {
	nodeName := "node" + conn.RemoteAddr().String()
	sharedFiles := parts[1:]

	if nodeInfo, exists := nodeInfoMap[nodeName]; exists {
		nodeInfo.SharedFiles = sharedFiles
		nodeInfoMap[nodeName] = nodeInfo
		sendResponse(conn, fmt.Sprintf("UPDATE_SUCCESS %s", nodeName))
	} else {
		sendResponse(conn, "ERROR Node não registrado")
	}
}

func handleLocate(conn net.Conn, parts []string) {
	if len(parts) < 2 {
		sendResponse(conn, "ERROR Comando LOCATE malformado")
		return
	}

	fileName := parts[1]

	locations := make([]string, 0)

	for nodeName, nodeInfo := range nodeInfoMap {
		if contains(nodeInfo.SharedFiles, fileName) {
			locations = append(locations, fmt.Sprintf("%s %s %s", nodeName, nodeInfo.Address, strings.Join(nodeInfo.SharedFiles, " ")))
		}
	}

	if len(locations) > 0 {
		sendResponse(conn, fmt.Sprintf("LOCATE_SUCCESS %s", strings.Join(locations, "\n")))
	} else {
		sendResponse(conn, "ERROR Arquivo não encontrado")
	}
}

func handleQuit(conn net.Conn) {
	//Tirar o FS_Node do mapa
	for nodeName, nodeInfo := range nodeInfoMap {
		if nodeInfo.Address == conn.RemoteAddr().String() {
			delete(nodeInfoMap, nodeName)
			fmt.Println("FS_Node desconectado:", nodeName)
			sendResponse(conn, "QUIT_SUCCESS")
			break
		}
	}
}

func contains(arr []string, item string) bool {
	for _, a := range arr {
		if a == item {
			return true
		}
	}
	return false
}

func sendResponse(conn net.Conn, response string) {
	conn.Write([]byte(response + "\n"))
}
