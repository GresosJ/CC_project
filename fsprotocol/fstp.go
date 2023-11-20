package fstp

import {
	"fmt"
	"net"
	"strings"
}

type DataBlock struct {
	BlockID string
	FileID string 				// Name? ou criar uma struct file com as infos?
	Data []byte
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


//Server(FS Node) -> Client
func sendDataBlock(clientAddress string, blockID string, fileID string) {
	
	// Client Addr
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

/*	
	ToAsk:
		Como vamos fzr em relacao ao sequencializacao dos ficheiros, para os transformar em blocos


	TODO:
		Mecanismo de timeout
		Formatacao do DataBlock

	TOSEE:
		Set(Read/Write)Buffe
*/