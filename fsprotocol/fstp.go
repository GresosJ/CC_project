package fstp

import {
	"fmt"
	"net"
	"strings"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
}

type DataBlock struct {
	BlockID string
	FileID string 				// Name? ou criar uma struct file com as infos?
	Data []byte
	Hash    string

}

// Client -> Server(FS Node)
func requestDataBlock(serverAddress string, blockID string, fileID string){
	
	conn := openUDPConn(serverAddress)
	defer conn.Close()

	//Formatar mensagem
	requestMessage := fmt.Printf("REQUEST %s %s", blockID, fileID)

	//Converte a mensagem para bytes
	data := []byte(message)

	sendUDPData(data, conn, "Erro ao enviar a mensagem UDP")

	//fmt.Println("MESSAGE SENT!")

} 


//Envia um pacote 
func sendDataBlock(clientAddress string, blockID string, fileID string, data []byte) {
	
	// Criar hash value atraves da data
	hash := calculateHash(data)

	datablock := DataBlock{
		BlockID: blockID,
		FileID: fileID,
		Data: data,
		Hash: hash,
	}

	//Converte a strut para bytes
	dbBytes, err := json.Marshal(datablock)
	if err != nil{
		fmt.Println("Erro ao converter Datablock para bytes", err)
		return
	}

	// Client Addr Abrimos uma conexao por block, mas podias abrir uma conexao e passar como argumento nesta funcao
	conn := openUDPConn(clientAddress)
	defer conn.Close()

	// TODO!
	// Datagrama vai ser igual ao UDP default

	sendUDPData(datablock, conn, "Erro no envio do datablock")

}

//Client -> Server
func confirmData(serverAddress string, blockID string, fileID string){

	conn := openUDPConn(serverAddress)
	defer conn.Close()

	confirmMessage := fmt.Printf("BLOCK_CONFIRMED %s %s", blockID, fileID)

	data := []byte(confirmMessage)

	sendUDPData(data, conn, "Erro no envio da confirmacao do Datablock")

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

func openUDPConn(addr string) *UDPConn{
	serverAddr, err := net.ResolveUDPAddr("udp", addr) 
	if err != nil {
		fmt.Println("Erro ao obter o IP:", err)
		return
	}

	//Abre ligacao UDP
	conn, err := net.DialUDP("udp",nil,addr)
	if err != nil {
		fmt.Println("Erro ao abrir ligacao UDP", err)
		return
	}

	return conn
}

func sendUDPData(data []byte, conn *UDPConn, errorMessage string){
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