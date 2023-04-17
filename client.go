package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

func clientSendMessageAndListen() {
	//Enable local listening on the client (mainly used to receive REPLY messages from nodes)
	go clientTcpListen()
	fmt.Printf("The client opens a listener at%s\n", clientAddr)

	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("|  PBFT test demo client has been entered, please start all nodes before sending messages！ :)  |")
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("Please enter the information to be deposited in the node below：")
	//First get the user input from the command line
	stdReader := bufio.NewReader(os.Stdin)
	for {
		data, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}
		r := new(Request)
		r.Timestamp = time.Now().UnixNano()
		r.ClientAddr = clientAddr
		r.Message.ID = getRandom()
		//The content of the message is the user's input
		r.Message.Content = strings.TrimSpace(data)
		br, err := json.Marshal(r)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(br))
		content := jointMessage(cRequest, br)
		//Default N0 is the master node, send the request information directly to N0
		tcpDial(content, nodeTable["N0"])
	}
}

// Returns a ten-digit random number as a msgid
func getRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := rand.Int(rand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}
