package dht

import (
	"encoding/hex"
	"net"
)

// Ping is the most basic query. "q" = "ping" A ping query has a single argument,
// "id" the value is a 20-byte string containing the senders node ID in network byte
// order. The appropriate response to a ping has a single key "id" containing the
// node ID of the responding node.
// arguments:  {"id" : "<querying nodes id>"}
// http://www.bittorrent.org/beps/bep_0005.html#ping
func (node *Node) Ping(addr *net.UDPAddr) error {
	return nil
}

// response: {"id" : "<queried nodes id>"}
func (node *Node) onPingQuery(query *KRPCQuery, addr *net.UDPAddr) error {
	response := KRPCResponse{
		Q:         PingType,
		QueriedID: node.ID,
	}
	data, err := response.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send %v response to %s:%d\n", response, addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}

// FindNode is used to find the contact information for a node given its ID.
// "q" == "find_node" A find_node query has two arguments, "id" containing
// the node ID of the querying node, and "target" containing the ID of the
// node sought by the queryer. When a node receives a find_node query, it
// should respond with a key "nodes" and value of a string containing the
// compact node info for the target node or the K (8) closest good nodes in
// its own routing table.
// arguments:  {"id" : "<querying nodes id>", "target" : "<id of target node>"}
// http://www.bittorrent.org/beps/bep_0005.html#find-node
func (node *Node) FindNode(addr *net.UDPAddr, nid []byte) error {
	req := KRPCQuery{
		T:         node.tokenManager.GenToken(),
		Q:         FindNodeType,
		NID:       node.ID,
		TargetNID: nid,
	}

	data, err := req.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send find_node query to %s:%d\n", addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}

// response: {"id" : "<queried nodes id>", "nodes" : "<compact node info>"}
func (node *Node) onFindNodeQuery(query *KRPCQuery, addr *net.UDPAddr) error {
	// var nodes = make([]*NodeInfo, 0, 8)
	// for i := 0; i < 8; i++ {
	// 	nodes = append(nodes, &NodeInfo{
	// 		ID:      GetNeighborNID(node.ID, query.TargetNID),
	// 		UDPAddr: node.UDPAddr,
	// 	})
	// }
	response := KRPCResponse{
		T:         query.T,
		Q:         FindNodeType,
		QueriedID: node.ID,
		Nodes:     make([]*NodeInfo, 0),
	}
	data, err := response.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send %v response to %s:%d\n", response, addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}

// GetPeers gets peers associated with a torrent infohash. "q" = "get_peers" A get_peers
// query has two arguments, "id" containing the node ID of the querying node,
// and "info_hash" containing the infohash of the torrent. If the queried node
// has peers for the infohash, they are returned in a key "values" as a list
// of strings. Each string containing "compact" format peer information for a
// single peer. If the queried node has no peers for the infohash, a key "nodes"
// is returned containing the K nodes in the queried nodes routing table closest
// to the infohash supplied in the query. In either case a "token" key is also
//  included in the return value. The token value is a required argument for a
// future announce_peer query. The token value should be a short binary string.
// arguments:  {"id" : "<querying nodes id>", "info_hash" : "<20-byte infohash of target torrent>"}
// http://www.bittorrent.org/beps/bep_0005.html#get-peers
func (node *Node) GetPeers(addr *net.UDPAddr, infoHash []byte) error {
	query := KRPCQuery{
		T:        node.tokenManager.GenToken(),
		NID:      node.ID,
		InfoHash: GenerateNodeID(20),
	}

	data, err := query.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send find_node query to %s:%d\n", addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}

// response: {"id" : "<queried nodes id>", "token" :"<opaque write token>", "values" : ["<peer 1 info string>", "<peer 2 info string>"]}
func (node *Node) onGetPeersQuery(query *KRPCQuery, addr *net.UDPAddr) error {
	response := KRPCResponse{
		T:         query.T,
		Q:         GetPeersType,
		QueriedID: GetNeighborNID(node.ID, query.InfoHash),
		Token:     string(query.InfoHash[:2]),
		Nodes:     make([]*NodeInfo, 0),
	}

	data, err := response.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send %s response to %s:%d\n", string(data), addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}

// AnnouncePeer announces that the peer, controlling the querying node, is downloading a torrent on a port.
// announce_peer has four arguments: "id" containing the node ID of the querying node, "info_hash" containing
// the infohash of the torrent, "port" containing the port as an integer, and the "token" received in response
// to a previous get_peers query. There is an optional argument called implied_port which value is either 0 or 1.
// If it is present and non-zero, the port argument should be ignored and the source port of the UDP packet
// should be used as the peer's port instead. This is useful for peers behind a NAT that may not know their
// external port, and supporting uTP, they accept incoming.
// arguments:  {"id" : "<querying nodes id>",
// 	"implied_port": <0 or 1>,
// 	"info_hash" : "<20-byte infohash of target torrent>",
// 	"port" : <port number>,
// 	"token" : "<opaque token>"}
// reference: http://www.bittorrent.org/beps/bep_0005.html#announce_peer
func (node *Node) AnnouncePeer(addr *net.UDPAddr, infoHash []byte, token string, impliedPort int8, port int) error {
	req := KRPCQuery{
		T:           node.tokenManager.GenToken(),
		Q:           AnnouncePeerType,
		NID:         node.ID,
		InfoHash:    infoHash,
		Token:       token,
		ImpliedPort: impliedPort,
		Port:        port,
	}

	data, err := req.Encode()
	if err != nil {
		return err
	}

	return node.writeToUdp(addr, data)
}

// response: {"id" : "<queried nodes id>"}
func (node *Node) onAnnouncePeer(query *KRPCQuery, addr *net.UDPAddr) error {

	port := query.Port
	if query.ImpliedPort == 1 {
		port = addr.Port
	}

	node.PeerHandler(addr.IP.String(), port,
		hex.EncodeToString(query.InfoHash),
		hex.EncodeToString(query.NID))

	response := KRPCResponse{
		T:         query.T,
		Q:         AnnouncePeerType,
		QueriedID: node.ID,
	}
	data, err := response.Encode()
	if err != nil {
		return err
	}

	// log.Printf("send %v response to %s:%d\n", response, addr.IP.String(), addr.Port)
	return node.writeToUdp(addr, data)
}
