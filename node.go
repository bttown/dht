package dht

import (
	"encoding/hex"
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"
)

var bootstrapNodes = []string{
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"router.utorrent.com:6881",
	"dht.libtorrent.org:25401",
}

var r = rand.New(rand.NewSource(time.Now().Unix()))

func GenerateNodeID(length int) []byte {
	buf := make([]byte, length)
	r.Read(buf)

	return buf
}

func GetNeighborNID(id, target []byte) []byte {
	buf := make([]byte, 0, len(id))
	buf = append(buf, target[:10]...)
	buf = append(buf, id[10:]...)
	return buf
}

type Node struct {
	NodeInfo
	localUDPAddr net.UDPAddr
	udpConn      *net.UDPConn
	NetWork      string
	tokenManager *TokenManager
	PeerHandler  func(ip string, port int, infoHash, peerID string)

	findNodeChan chan *NodeInfo
	stopJoinDHT  chan bool
}

func NewNode(opts ...NodeOption) *Node {
	nid, _ := hex.DecodeString("f0fbf054cc37063b8b773d5dfaf7ebc84e83dce0")
	// nid := GenerateNodeID(20)
	node := Node{
		NodeInfo: NodeInfo{
			ID: nid,
		},
		NetWork: "udp",

		tokenManager: defaultTokenManager,

		findNodeChan: make(chan *NodeInfo, 100),
		stopJoinDHT:  make(chan bool),
	}

	for _, option := range opts {
		option(&node)
	}

	return &node
}

func (node *Node) joinDHTNetwork() error {

	go func() {
		for newNode := range node.findNodeChan {
			node.FindNode(&newNode.UDPAddr, GenerateNodeID(20))
		}
	}()

	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-node.stopJoinDHT:
			return nil
		case <-ticker:
			for _, bootStrapNode := range bootstrapNodes {
				nodeAddr, err := net.ResolveUDPAddr(node.NetWork, bootStrapNode)
				if err != nil {
					continue
				}
				node.FindNode(nodeAddr, GenerateNodeID(20))
			}
		}
	}
}

func (node *Node) handleKRPCMsg(srcAddr *net.UDPAddr, b []byte) error {
	defer func() {
		if r := recover(); r != nil {
			var buf = make([]byte, 1024)
			n := runtime.Stack(buf, false)
			log.Println("recover", r, "/n", string(buf[:n]), "data", b)
		}
	}()

	msg, err := NewKRPCMessage(b)
	if err != nil {
		log.Println("NewKRPCMessage fatal", err)
		return err
	}

	// handle krpc query message.
	if msg.IsQuery() {
		query := new(KRPCQuery)
		query.Loads(msg.data)

		switch query.Q {
		case PingType:
			node.onPingQuery(query, srcAddr)
		case FindNodeType:
			node.onFindNodeQuery(query, srcAddr)
		case GetPeersType:
			node.onGetPeersQuery(query, srcAddr)
		case AnnouncePeerType:
			node.onAnnouncePeer(query, srcAddr)
		default:
			return nil
		}
	} else if msg.IsResponse() {
		response := new(KRPCResponse)
		response.Loads(msg.data)

		if len(response.Nodes) > 0 {
			for _, nodeInfo := range response.Nodes {
				node.findNodeChan <- nodeInfo
			}
		}
	}

	return nil
}

func (node *Node) writeToUdp(addr *net.UDPAddr, data []byte) error {
	node.udpConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := node.udpConn.WriteToUDP(data, addr)
	if err != nil {
		return err
	}

	if n != len(data) {
		return errors.New("udp write uncompleted")
	}
	return err
}

func (node *Node) receiveUDP(conn *net.UDPConn) error {
	var buf = make([]byte, 8192)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		newBuf := make([]byte, n)
		copy(newBuf, buf)
		if len(newBuf) != n {
			panic("copy buffer fatal")
		}
		go node.handleKRPCMsg(addr, newBuf)
	}
}

func (node *Node) serveUDP() error {

	conn, err := net.ListenUDP(node.NetWork, &node.localUDPAddr)
	if err != nil {
		return err
	}

	go func() {
		err := node.receiveUDP(conn)
		log.Println("quit receiveUDP with", err)
	}()
	node.udpConn = conn
	return nil
}

func (node *Node) WaitSignal() error {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("stop node...")
	close(node.findNodeChan)
	node.stopJoinDHT <- true
	close(node.stopJoinDHT)

	return node.udpConn.Close()
}

func (node *Node) Serve(opts ...NodeOption) error {
	for _, opt := range opts {
		opt(node)
	}

	log.Printf("start node %s...\n", node.NodeInfo.String())
	log.Printf("Lo address => %s:%d", node.localUDPAddr.IP.String(), node.localUDPAddr.Port)
	log.Printf("WAN address => %s:%d", node.UDPAddr.IP.String(), node.UDPAddr.Port)
	if err := node.serveUDP(); err != nil {
		log.Println("start udp listener fatal", err)
		return err
	}

	log.Println("start udp listener...")

	go node.joinDHTNetwork()

	log.Println("start to join the dht network...")

	return node.WaitSignal()
}
