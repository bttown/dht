package dht

import (
	"bytes"
	// "encoding/hex"
	"errors"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/bttown/routing-table"
)

var bootstrapNodes = []string{
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"router.utorrent.com:6881",
	"dht.libtorrent.org:25401",
}

var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateBytes() []byte {
	buf := make([]byte, NodeIDBytes)
	randSource.Read(buf)
	return buf
}

func GenerateNodeID() NodeID {
	buf := generateBytes()
	var nid NodeID
	copy(nid[:], buf[:])
	return nid
}

func GetNeighborNID(id NodeID, hash []byte) NodeID {
	// Fix bug: when quering node id is empty, it will cause panic
	if len(hash) == 0 {
		return id
	}
	buf := make([]byte, 0, len(id))
	buf = append(buf, hash[:10]...)
	buf = append(buf, id[10:]...)

	var nid NodeID
	copy(nid[:], buf[:])
	return nid
}

type Node struct {
	NodeInfo
	localUDPAddr net.UDPAddr
	udpConn      *net.UDPConn
	NetWork      string
	tokenManager *TokenManager
	table        *table.Table
	PeerHandler  func(ip string, port int, infoHash, peerID string)

	findNodeChan chan *NodeInfo
	closed       chan struct{}
	dumpFileName string
	running      bool
}

func NewNode(opts ...NodeOption) *Node {
	node := &Node{
		NodeInfo:     NodeInfo{},
		NetWork:      "udp",
		dumpFileName: "dump.ktb",

		tokenManager: defaultTokenManager,

		findNodeChan: make(chan *NodeInfo, 300),
		closed:       make(chan struct{}),
	}

	b := GenerateNodeID()
	copy(node.ID[:], b[:])
	t := table.NewTable(table.Hash(node.ID), node)
	tid := t.OwnerID()
	if !bytes.Equal(tid[:], node.ID[:]) {
		copy(node.ID[:], tid[:])
	}

	node.table = t

	for _, option := range opts {
		option(node)
	}

	return node
}

func (node *Node) joinDHTNetwork() error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		id := GenerateNodeID()
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

			neighbors := node.table.Closest(table.Hash(id), 8)

			for _, neighbor := range neighbors.Entries() {
				// log.Println("send find node to neighbor node", neighbor)
				node.FindNode(&neighbor.UDPAddr, id)
			}
		}
	}
}

func (node *Node) handleKRPCMsg(remote *net.UDPAddr, b []byte) error {
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

		contactID := table.Hash(query.NID)
		node.table.Update(&table.Contact{
			UDPAddr: *remote,
			NID:     contactID,
		})

		switch query.Q {
		case PingType:
			node.onPingQuery(query, remote)
		case FindNodeType:
			node.onFindNodeQuery(query, remote)
		case GetPeersType:
			node.onGetPeersQuery(query, remote)
		case AnnouncePeerType:
			node.onAnnouncePeer(query, remote)
		default:
			return nil
		}

	} else if msg.IsResponse() {
		r := new(KRPCResponse)
		r.Loads(msg.data)

		contactID := table.Hash(r.QueriedID)
		node.table.Update(&table.Contact{
			UDPAddr: *remote,
			NID:     contactID,
		})

		if len(r.Nodes) > 0 {
			for _, nodeInfo := range r.Nodes {
				node.findNodeChan <- nodeInfo
			}
		}
	} else {
		log.Println("unknown krpc msg", msg)
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

	node.table.Stop()

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
		log.Println("start UDP listener fatal", err)
		return err
	}

	log.Println("start UDP listener...")

	go node.joinDHTNetwork()

	log.Println("start to join the DHT network...")

	return node.WaitSignal()
}
