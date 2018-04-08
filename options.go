package dht

import (
	"log"
	"net"
)

type NodeOption func(node *Node)

func OptionAddress(addr string) NodeOption {
	return func(node *Node) {
		addr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			panic(err)
		}

		log.Println("quering my ip addr...")
		ip, err := GetMyIP()
		if err != nil {
			panic(err)
		}

		log.Println(ip)

		node.UDPAddr = net.UDPAddr{
			IP:   net.ParseIP(ip),
			Port: addr.Port,
		}
		node.localUDPAddr = *addr
	}
}
