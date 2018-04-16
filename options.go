package dht

import (
	"encoding/hex"
	"net"
)

const (
	RANDOM = "random"
)

type NodeOption func(node *Node)

func OptionNodeID(nid string) NodeOption {
	return func(node *Node) {
		if nid == RANDOM {
			node.ID = GenerateNodeID(20)
		} else {
			b, err := hex.DecodeString(nid)
			if err != nil || len(b) != 20 {
				panic("Invalid node ID")
			}

			node.ID = b
		}
	}
}

func OptionAddress(addr string) NodeOption {
	return func(node *Node) {
		addr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			panic(err)
		}

		// log.Println("quering my ip addr...")
		// ip, err := GetMyIP()
		// if err != nil {
		// 	panic(err)
		// }

		// log.Println(ip)

		// node.UDPAddr = net.UDPAddr{
		// 	IP:   net.ParseIP(ip),
		// 	Port: addr.Port,
		// }
		node.localUDPAddr = *addr
	}
}
