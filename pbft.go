package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
)

// Local message pool (simulates the persistence layer), which is only deposited after a successful commit is confirmed
var localMessagePool = []Message{}

type node struct {
	//Node ID
	nodeID string
	//Node Listening Address
	addr string
	//RSA Private Key
	rsaPrivKey []byte
	//RSA Public Key
	rsaPubKey []byte
}

type pbft struct {
	//Node Information
	node node
	//Self-increasing serial number per request
	sequenceID int
	//Lock
	lock sync.Mutex
	//Temporary message pool with message digest corresponding to the message ontology
	messagePool map[string]Request
	//Store the number of prepares received (at least 2f need to be received and confirmed), corresponding to the summary
	prePareConfirmCount map[string]map[string]bool
	//Store the number of commissions received (at least 2f+1 need to be received and acknowledged), corresponding to the summary
	commitConfirmCount map[string]map[string]bool
	//Whether the message has been Commit broadcast
	isCommitBordcast map[string]bool
	//Whether the message has been Reply to the client
	isReply map[string]bool
}

func NewPBFT(nodeID, addr string) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = p.getPivKey(nodeID) //Read from the generated private key file
	p.node.rsaPubKey = p.getPubKey(nodeID)  //Read from the generated private key file
	p.sequenceID = 0
	p.messagePool = make(map[string]Request)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	return p
}

func (p *pbft) handleRequest(data []byte) {
	//Cutting messages, calling different functions according to the message command
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case cRequest:
		p.handleClientRequest(content)
	case cPrePrepare:
		p.handlePrePrepare(content)
	case cPrepare:
		p.handlePrepare(content)
	case cCommit:
		p.handleCommit(content)
	}
}

// Processing requests from clients
func (p *pbft) handleClientRequest(content []byte) {
	fmt.Println("The master node has received the request from the client ...")
	//Parsing out Request structures using json
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//Add information serial number
	p.sequenceIDAdd()
	//Get Message Summary
	digest := getDigest(*r)
	fmt.Println("The request has been stored in a temporary message pool")
	//Store in temporary message pool
	p.messagePool[digest] = *r
	//Signature of message digest by master node
	digestByte, _ := hex.DecodeString(digest)
	signInfo := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
	//Splice into PrePrepare, ready to send to follower node
	pp := PrePrepare{*r, digest, p.sequenceID, signInfo}
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("PrePrepare broadcast to other nodes is in progress ...")
	//Perform PrePrepare Broadcast
	p.broadcast(cPrePrepare, b)
	fmt.Println("PrePrepare broadcast completion")
}

// Processing pre-prepared messages
func (p *pbft) handlePrePrepare(content []byte) {
	fmt.Println("This node has received the PrePrepare from the master node ...")
	//	//Parsing out the PrePrepare structure using json
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}
	//Obtain the public key of the master node for digital signature verification
	primaryNodePubKey := p.getPubKey("N0")
	digestByte, _ := hex.DecodeString(pp.Digest)
	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("The message summary does not match, and the PREPARE broadcast is rejected.")
	} else if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("The message serial number does not match, and the prepare broadcast is rejected.")
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		fmt.Println("Master node signature verification failed! , Prepare broadcast rejected")
	} else {
		//Serial number assignment
		p.sequenceID = pp.SequenceID
		//Store messages in a temporary message pool
		fmt.Println("Messages have been stored in the temporary node pool")
		p.messagePool[pp.Digest] = pp.RequestMessage
		//The node uses the private key to sign it
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//Splice into Prepare
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID, sign}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//Conducting the preparatory phase of broadcasting
		fmt.Println("Prepare broadcast in progress ...")
		p.broadcast(cPrepare, bPre)
		fmt.Println("Prepare broadcast completion")
	}
}

// Handling preparation messages
func (p *pbft) handlePrepare(content []byte) {
	//Parsing out the Prepare structure using json
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("This node has received the Prepare from %s node ... \n", pre.NodeID)
	//Obtain the public key of the message source node for digital signature verification
	MessageNodePubKey := p.getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("The current temporary message pool does not have this summary and refuses to execute the commit broadcast")
	} else if p.sequenceID != pre.SequenceID {
		fmt.Println("The message serial number does not match and the commit broadcast is rejected")
	} else if !p.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		fmt.Println("Node signature verification failed! Refusal to execute commit broadcast")
	} else {
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//Because the master node does not send Prepare, it does not include itself
		specifiedCount := 0
		if p.node.nodeID == "N0" {
			specifiedCount = nodeCount / 3 * 2
		} else {
			specifiedCount = (nodeCount / 3 * 2) - 1
		}
		//If the node has received at least 2f prepare messages (including itself) and has not broadcasted a commit, then it will broadcast a commit
		p.lock.Lock()
		//Obtain the public key of the message source node for digital signature verification
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			fmt.Println("This node has received the Prepare message from at least 2f nodes (including local nodes) ...")
			//The node uses the private key to sign it
			sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			c := Commit{pre.Digest, pre.SequenceID, p.node.nodeID, sign}
			bc, err := json.Marshal(c)
			if err != nil {
				log.Panic(err)
			}
			//Broadcast of submission information
			fmt.Println("Commit broadcast in progress")
			p.broadcast(cCommit, bc)
			p.isCommitBordcast[pre.Digest] = true
			fmt.Println("commit broadcast completed")
		}
		p.lock.Unlock()
	}
}

// Handling submission confirmation messages
func (p *pbft) handleCommit(content []byte) {
	//Parsing out the Commit structure using json
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("This node has received the Commit from %s node ... \n", c.NodeID)
	//Obtain the public key of the message source node for digital signature verification
	MessageNodePubKey := p.getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		fmt.Println("The current prepare pool does not have this summary and refuses to persist the message to the local message pool")
	} else if p.sequenceID != c.SequenceID {
		fmt.Println("Message serial number does not match, refusing to persist the message to the local message pool")
	} else if !p.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		fmt.Println("Node signature verification failed! Message persistence to local message pool rejected")
	} else {
		p.setCommitConfirmMap(c.Digest, c.NodeID, true)
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		// If the node has received at least 2f+1 commit messages (including itself), and the node has not replied, and a commit broadcast has been made, the message is submitted to the local message pool, and the success of the reply is flagged to the client!
		p.lock.Lock()
		if count >= nodeCount/3*2 && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			fmt.Println("This node has received Commit messages from at least 2f + 1 nodes (including local nodes) ...")
			//Submit the message information, to the local message pool!
			localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
			info := p.node.nodeID + "The node has set msgid." + strconv.Itoa(p.messagePool[c.Digest].ID) + "The message is stored in the local message pool, and the message content is:" + p.messagePool[c.Digest].Content
			fmt.Println(info)
			fmt.Println("Being reply client ...")
			tcpDial([]byte(info), p.messagePool[c.Digest].ClientAddr)
			p.isReply[c.Digest] = true
			fmt.Println("End of reply")
		}
		p.lock.Unlock()
	}
}

// Serial number accumulation
func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

// Broadcast to nodes other than yourself
func (p *pbft) broadcast(cmd command, content []byte) {
	for i := range nodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := jointMessage(cmd, content)
		go tcpDial(message, nodeTable[i])
	}
}

// Assigning values to multiple mapping openings
func (p *pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

// Assigning values to multiple mapping openings
func (p *pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

// Pass in the node number and get the corresponding public key
func (p *pbft) getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// Pass in the node number and get the corresponding private key
func (p *pbft) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}
