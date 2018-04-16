package dht

import (
	// "encoding/hex"
	"errors"
	"github.com/bttown/kbucket"
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

func GenerateNodeID(length int) []byte {
	var r = rand.New(rand.NewSource(time.Now().UnixNano()))
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
	routeTable   *kbucket.RouteTable
	PeerHandler  func(ip string, port int, infoHash, peerID string)

	findNodeChan chan *NodeInfo
	closed       chan struct{}
	dumpFileName string
	running      bool
}

func NewNode(opts ...NodeOption) *Node {
	node := Node{
		NodeInfo:     NodeInfo{},
		NetWork:      "udp",
		dumpFileName: "dump.ktb",

		tokenManager: defaultTokenManager,

		findNodeChan: make(chan *NodeInfo, 300),
		closed:       make(chan struct{}),
	}

	routeTable, err := kbucket.NewFromDumpFile(node.dumpFileName)
	if err != nil {
		// node.ID = hex.DecodeString("f0fbf054cc37063b8b773d5dfaf7ebc84e83dce0")
		node.ID = GenerateNodeID(20)
		node.routeTable = kbucket.New(node.ID)
	} else {
		node.ID = routeTable.OwnerID()
		node.routeTable = routeTable
	}

	for _, option := range opts {
		option(&node)
	}

	return &node
}

func (node *Node) bgSave() {
	ticker := time.Tick(25 * time.Second)
	for {
		<-ticker
		f, err := os.Create(node.dumpFileName)
		if err != nil {
			log.Println("save dump.tb failed", err)
			return
		}

		err = node.routeTable.Dump(f)
		if err != nil {
			log.Println("save dump.tb failed", err)
		}
		f.Close()
	}
}

func (node *Node) joinDHTNetwork() error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		id := node.GetID()
		select {
		case <-node.closed:
			return nil
		case info := <-node.findNodeChan:
			node.FindNode(&info.UDPAddr, id)
		case <-ticker.C:
			for _, bootStrapNode := range bootstrapNodes {
				nodeAddr, err := net.ResolveUDPAddr(node.NetWork, bootStrapNode)
				if err != nil {
					continue
				}
				node.FindNode(nodeAddr, id)
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

		node.routeTable.Add(kbucket.Contact{
			ID:      query.NID,
			UDPAddr: *srcAddr,
		})

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
	} else {
		log.Println(msg)
	}

	return nil
}

func (node *Node) writeToUDP(addr *net.UDPAddr, data []byte) error {
	node.udpConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := node.udpConn.WriteToUDP(data, addr)
	if err != nil {
		log.Println("writeToUdp", err)
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
	close(node.closed)

	node.running = false
	return node.udpConn.Close()
}

func (node *Node) Serve(opts ...NodeOption) error {
	for _, opt := range opts {
		opt(node)
	}

	node.running = true

	log.Printf("start node %s...\n", node.NodeInfo.String())
	log.Printf("Lo address => %s:%d", node.localUDPAddr.IP.String(), node.localUDPAddr.Port)
	log.Printf("WAN address => %s:%d", node.UDPAddr.IP.String(), node.UDPAddr.Port)
	if err := node.serveUDP(); err != nil {
		log.Println("start udp listener fatal", err)
		return err
	}

	log.Println("start udp listener...")

	go node.joinDHTNetwork()
	go node.bgSave()

	log.Println("start to join the dht network...")

	return node.WaitSignal()
}
