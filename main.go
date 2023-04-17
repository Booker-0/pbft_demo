package main

import (
	"log"
	"os"
)

const nodeCount = 4

// Listening address of the client
var clientAddr = "127.0.0.1:8888"

// Node pool, mainly used to store listening addresses
var nodeTable map[string]string

func main() {
	//Generate public and private keys for the four test nodes
	genRsaKeys()
	nodeTable = map[string]string{
		"N0": "127.0.0.1:8000",
		"N1": "127.0.0.1:8001",
		"N2": "127.0.0.1:8002",
		"N3": "127.0.0.1:8003",
	}
	if len(os.Args) != 2 {
		log.Panic("The input parameter is wrong！")
	}
	nodeID := os.Args[1]
	if nodeID == "client" {
		clientSendMessageAndListen() //Start the client program
	} else if addr, ok := nodeTable[nodeID]; ok {
		p := NewPBFT(nodeID, addr)
		go p.tcpListen() //Start Node
	} else {
		log.Fatal("No such node number!")
	}
	select {}
}
