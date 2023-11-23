package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

type DataBlock struct {
	BlockID string
	FileID  string // Name? ou criar uma struct file com as infos?
	Data    []byte
	Hash    string
}

// Client -> Server(FS Node)
func requestDataBlock(serverAddress string, blockID string, fileID string) {

	conn, err := openUDPConn(serverAddress)
	if err != nil {
		fmt.Println("Erro ao abrir a conexão UDP:", err)
		return
	}
	defer conn.Close()

	// Format the message
	requestMessage := fmt.Sprintf("REQUEST %s %s", blockID, fileID)

	// Convert the message to bytes
	data := []byte(requestMessage)

	sendUDPData(data, *conn, "Erro ao enviar a mensagem UDP")

	//fmt.Println("MESSAGE SENT!")

}

// Envia um pacote
func sendDataBlock(clientAddress string, blockID string, fileID string, data []byte) {

	// Criar hash value atraves da data
	hash := calculateHash(data)

	datablock := DataBlock{
		BlockID: blockID,
		FileID:  fileID,
		Data:    data,
		Hash:    hash,
	}

	//Converte a strut para bytes
	dbBytes, err := json.Marshal(datablock)
	if err != nil {
		fmt.Println("Erro ao converter Datablock para bytes", err)
		return
	}

	// Client Addr Abrimos uma conexao por block, mas podias abrir uma conexao e passar como argumento nesta funcao
	conn, err := openUDPConn(clientAddress)
	if err != nil {
		fmt.Println("Erro ao abrir a conexão UDP:", err)
		return
	}
	defer conn.Close()

	// TODO!
	// Datagrama vai ser igual ao UDP default

	sendUDPData(dbBytes, *conn, "Erro no envio do datablock")

}

// Client -> Server
func confirmData(serverAddress string, blockID string, fileID string) {

	conn, err := openUDPConn(serverAddress)
	if err != nil {
		fmt.Println("Erro ao abrir a conexão UDP:", err)
		return
	}
	defer conn.Close()

	confirmMessage := fmt.Sprintf("BLOCK_CONFIRMED %s %s", blockID, fileID)

	data := []byte(confirmMessage)

	sendUDPData(data, *conn, "Erro no envio da confirmacao do Datablock")

}

func checkReceivedDataBlock(data []byte) {

	// Decodifica a estrutura DataBlock
	var datablock DataBlock
	err := json.Unmarshal(data, &datablock)
	if err != nil {
		fmt.Println("Erro ao decodificar o DataBlock", err)
		return
	}

	// Calcula o hash dos dados recebidos
	receivedHash := calculateHash(datablock.Data)

	// Compara os hashes codes
	if receivedHash == datablock.Hash {
		fmt.Println("Integridade verificada. Hashes coincidem.")
		// Continue o processamento dos dados conforme necessário
	} else {
		fmt.Println("Erro: Integridade comprometida. Hashes não coincidem.")
		// Manipule a situação de integridade comprometida conforme necessário
	}
}

//////////////////// Utils Functions ////////////////////

func openUDPConn(addr string) (*net.UDPConn, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Erro ao obter o IP:", err)
		return nil, err
	}

	// Abre ligacao UDP
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Erro ao abrir ligacao UDP", err)
		return nil, err
	}

	return conn, nil
}

func sendUDPData(data []byte, conn net.UDPConn, errorMessage string) {
	_, err := conn.Write(data)
	if err != nil {
		fmt.Println(errorMessage, err)
		return
	}
}

func calculateHash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash
}

/*
	ToAsk:
		Como vamos fzr em relacao ao sequencializacao dos ficheiros, para os transformar em blocos


	TODO:
		Mecanismo de timeout
		Formatacao do DataBlock

	TOSEE:
		Set(Read/Write)Buffe
*/
