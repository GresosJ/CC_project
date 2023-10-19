package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Estrutura para representar informações sobre um FS_Node
type FSNodeInfo struct {
	Address     string
	SharedFiles []string
}

// Mapeamento de nomes de FS_Node para informações
var nodeInfoMap map[string]FSNodeInfo

func main() {
	nodeInfoMap = make(map[string]FSNodeInfo)

	// Porta em que o servidor FS_Tracker escutará
	port := 9090
	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	}

	// Inicie o servidor FS_Tracker na porta especificada
	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(port))
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
	fmt.Println("Client connected:", conn.RemoteAddr())

	// ISTO É SO PARA VER SE O CLIENTE CONSEGUE SE CONECTAR AO SERVIDOR (É UM TESTE)
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

	// Aqui, você deve implementar a lógica de manipulação de mensagens FS Track Protocol.
	// Isso incluirá o registro de um FS_Node, a atualização da lista de arquivos/blocos
	// e a resposta a pedidos de localização de arquivos.

	// Por exemplo, você pode usar bufio.NewReader para ler mensagens do cliente e responder
	// de acordo com o protocolo.

	// Exemplo:
	// reader := bufio.NewReader(conn)
	// message, _ := reader.ReadString('\n')
	// message = strings.TrimSpace(message)

	// Implemente o código para processar mensagens FS Track Protocol aqui.

	// Exemplo de registro de um FS_Node:
	// if strings.HasPrefix(message, "REGISTER") {
	//     parts := strings.Split(message, " ")
	//     nodeName := parts[1]
	//     address := parts[2]
	//     files := parts[3:]
	//     // Registre o FS_Node no nodeInfoMap
	//     // ...
	// }

	// Exemplo de pedido de localização de um arquivo:
	// if strings.HasPrefix(message, "LOCATE") {
	//     // Processar o pedido de localização de arquivo e responder com a lista de FS_Node
	//     // que possuem o arquivo
	//     // ...
	// }
}
