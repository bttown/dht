package dht

import (
	"errors"

	"github.com/IncSW/go-bencode"
)

type QueryType string

var (
	PingType         QueryType = "ping"
	FindNodeType     QueryType = "find_node"
	GetPeersType     QueryType = "get_peers"
	AnnouncePeerType QueryType = "announce_peer"
)

var ErrUnKnowQueryType = errors.New("Unknow query type")

type KRPCMessage struct {
	T    string
	Y    string
	data map[string]interface{}
}

func NewKRPCMessage(b []byte) (*KRPCMessage, error) {
	in, err := bencode.Unmarshal(b)
	if err != nil {
		return nil, err
	}

	msg := new(KRPCMessage)
	data := in.(map[string]interface{})
	if t, ok := data["t"].([]byte); ok {
		msg.T = string(t)
	}
	if y, ok := data["y"].([]byte); ok {
		msg.Y = string(y)
	}
	msg.data = data
	return msg, err
}

func (msg *KRPCMessage) IsQuery() bool {
	return msg.Y == "q"
}

func (msg *KRPCMessage) IsResponse() bool {
	return msg.Y == "r"
}

func (msg *KRPCMessage) IsError() bool {
	return msg.Y == "e"
}

type KRPCResponse struct {
	T         []byte
	Q         QueryType
	QueriedID NodeID
	Token     string
	Nodes     []*NodeInfo
	Values    []string
}

func (resp *KRPCResponse) Loads(data map[string]interface{}) error {
	resp.T = data["t"].([]byte)
	data = data["r"].(map[string]interface{})
	if queriedID, ok := data["id"]; ok {
		copy(resp.QueriedID[:], queriedID.([]byte))
	}
	if token, ok := data["token"]; ok {
		resp.Token = string(token.([]byte))
	}
	if nodes, ok := data["nodes"]; ok {
		resp.Nodes = UnCompactNodeInfos(nodes.([]byte))
	}
	if values, ok := data["values"]; ok {
		vs := values.([]interface{})
		for _, v := range vs {
			resp.Values = append(resp.Values, string(v.([]byte)))
		}
	}

	return nil
}

func (resp *KRPCResponse) Encode() ([]byte, error) {
	data := map[string]interface{}{
		"t": []byte(resp.T),
		"y": []byte("r"),
	}

	switch resp.Q {
	case PingType:
		data["r"] = map[string]interface{}{
			"id": resp.QueriedID[:],
		}
	case FindNodeType:
		data["r"] = map[string]interface{}{
			"id":    resp.QueriedID[:],
			"nodes": CompactNodeInfos(resp.Nodes),
		}
	case GetPeersType:
		data["r"] = map[string]interface{}{
			"id":    resp.QueriedID[:],
			"token": resp.Token,
			"nodes": CompactNodeInfos(resp.Nodes),
		}
	case AnnouncePeerType:
		data["r"] = map[string]interface{}{
			"id": resp.QueriedID[:],
		}
	default:
		return nil, ErrUnKnowQueryType
	}
	return bencode.Marshal(data)
}

type KRPCQuery struct {
	T []byte // krpc query token
	Q QueryType

	NID         NodeID
	TargetNID   NodeID
	InfoHash    []byte
	ImpliedPort int8
	Port        int
	Token       string
}

func LoadKRPCErrorMsg(data map[string]interface{}) error {
	if e, ok := data["e"]; ok {
		return errors.New(string(e.([]interface{})[1].([]byte)))
	}

	return nil
}

func (query *KRPCQuery) Loads(data map[string]interface{}) error {
	if t, ok := data["t"]; ok {
		query.T = t.([]byte)
	}

	if queryType, ok := data["q"]; ok {
		query.Q = QueryType(string(queryType.([]byte)))
	}

	data = data["a"].(map[string]interface{})

	if nid, ok := data["id"]; ok {
		copy(query.NID[:], nid.([]byte))
	}

	if target, ok := data["target"]; ok {
		copy(query.TargetNID[:], target.([]byte))
	}

	if infoHash, ok := data["info_hash"]; ok {
		query.InfoHash = infoHash.([]byte)
	}

	if impliedPort, ok := data["implied_port"]; ok {
		query.ImpliedPort = int8(impliedPort.(int64))
	}

	if port, ok := data["port"]; ok {
		query.Port = int(port.(int64))
	}

	if token, ok := data["token"]; ok {
		query.Token = string(token.([]byte))
	}

	return nil
}

func (query *KRPCQuery) Encode() ([]byte, error) {
	data := map[string]interface{}{
		"t": []byte(query.T),
		"y": []byte("q"),
		"q": []byte(query.Q),
	}

	switch query.Q {
	case PingType:
		data["a"] = map[string]interface{}{
			"id": query.NID[:],
		}
	case FindNodeType:
		data["a"] = map[string]interface{}{
			"id":     query.NID[:],
			"target": query.TargetNID[:],
		}
	case GetPeersType:
		data["a"] = map[string]interface{}{
			"id":        query.NID[:],
			"info_hash": query.InfoHash,
		}
	case AnnouncePeerType:
		data["a"] = map[string]interface{}{
			"id":           query.NID[:],
			"implied_port": query.ImpliedPort,
			"info_hash":    query.InfoHash,
			"port":         query.Port,
			"token":        query.Token,
		}
	default:
		return nil, ErrUnKnowQueryType
	}
	return bencode.Marshal(data)
}
