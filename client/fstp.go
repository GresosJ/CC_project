package main

import (
	/* "bytes" */
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

const hashSize = 32    // 32 bytes
const fileIDSize = 54  // 54 byes
const blockIDSize = 11 // ate 99GB
const maxBlockSize = 1472 - hashSize - fileIDSize - blockIDSize - len("\"BlockID\":\"\"") - len("\"FileID\":\"\"") - len("\"Data\":null") - len("\"Hash\":\"\"") - 23 - 324
const totalBlockSize = 1472

type DataBlock struct {
	BlockID string
	FileID  string
	Data    []byte
	Hash    string
}

// Client -> Server(FS Node)
func requestDataBlock(conn *net.UDPConn, blockID string, fileID string) {

	// Format the message
	requestMessage := fmt.Sprintf("REQUEST %s %s", blockID, fileID)

	// Convert the message to bytes
	data := []byte(requestMessage)

	sendUDPData(data, *conn, "Erro ao enviar a mensagem UDP")

	//fmt.Println("MESSAGE SENT!")

}

// Envia um pacote
func sendDataBlock(conn *net.UDPConn, addr *net.UDPAddr, blockID string, fileID string, data []byte) {

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

	if len(dbBytes) > totalBlockSize {
        fmt.Println("Erro: Dados excedem o tamanho máximo permitido ", len(dbBytes), " " , totalBlockSize)
        return
    }

	/* dbBytes = append(dbBytes, []byte("\nEND\n")...) */

	_, err = conn.WriteToUDP(dbBytes, addr)
	if err != nil {
		fmt.Println("Erro no envio do datablock", err)
		return
	}

}

// Client -> Server
func confirmData(conn *net.UDPConn, blockID string, fileID string) {

	confirmMessage := fmt.Sprintf("BLOCK_CONFIRMED %s %s", blockID, fileID)

	data := []byte(confirmMessage)

	sendUDPData(data, *conn, "Erro no envio da confirmacao do Datablock")

}

func checkReceivedDataBlock(data []byte) []byte {
	/* // Encontra o índice do marcador de fim
	endMarkerIndex := bytes.Index(data, []byte("\nEND\n"))
	if endMarkerIndex == -1 {
		//fmt.Println("Marcador de fim não encontrado")
		return nil
	} */

	/* // Extrai os dados JSON
	jsonBytes := data[:endMarkerIndex]

	// Calcula o hash dos dados recebidos
	receivedHash := calculateHash(data[:endMarkerIndex]) */

	jsonBytes := data
	
	// Compara os hashes codes
	var datablock DataBlock
	err := json.Unmarshal(jsonBytes, &datablock)
	if err != nil {
		fmt.Println("Erro ao decodificar o DataBlock", err)
		return nil
	}
	
	receivedHash := calculateHash(datablock.Data)
	if receivedHash == datablock.Hash {
		fmt.Println("Integridade verificada. Hashes coincidem.")
		return datablock.Data
		// Continue o processamento dos dados conforme necessário
	} else {
		fmt.Println("Erro: Integridade comprometida. Hashes não coincidem.")
		return nil
		// Manipule a situação de integridade comprometida conforme necessário
	}
}

//////////////////// Utils Functions ////////////////////

func openUDPConn(addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Erro ao obter o IP", err)
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
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
