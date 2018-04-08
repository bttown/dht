package dht

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
)

const NodeInfoEncodedLength = 26

type NodeInfo struct {
	ID []byte
	net.UDPAddr
}

func (info *NodeInfo) String() string {
	return fmt.Sprintf("<node-info nid:%s ip:%s port:%d>", hex.EncodeToString(info.ID), info.IP.String(), info.Port)
}

func CompactNodeInfos(nodes []*NodeInfo) []byte {
	var data = make([]byte, 0, NodeInfoEncodedLength*len(nodes))
	var portBuff = make([]byte, 2)
	var ipBuff = make([]byte, 4)
	for _, node := range nodes {
		if len(node.IP) == net.IPv4len {
			ipBuff = []byte(node.IP)
		} else if len(node.IP) == net.IPv6len {
			ipBuff = []byte(node.IP[12:16])
		} else {
			continue
		}
		binary.LittleEndian.PutUint16(portBuff, uint16(node.Port))
		data = append(data, node.ID...)
		data = append(data, ipBuff...)
		data = append(data, portBuff...)
	}

	return data
}

func UnCompactNodeInfos(b []byte) []*NodeInfo {

	length := len(b)
	if length%NodeInfoEncodedLength != 0 {
		return nil
	}

	var infos = make([]*NodeInfo, 0, length/NodeInfoEncodedLength)
	for i := 0; i < length; i += NodeInfoEncodedLength {
		ndInfo := &NodeInfo{
			ID: b[i : i+20],
			UDPAddr: net.UDPAddr{
				IP:   net.IPv4(b[i+20], b[i+21], b[i+22], b[i+23]),
				Port: int(binary.LittleEndian.Uint16(b[i+24 : i+26])),
			},
		}

		infos = append(infos, ndInfo)
	}

	return infos
}
