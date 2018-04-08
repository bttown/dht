# dht
golang dht(Distributed Hash Table) node

#### Install

    go get -u github.com/bttown/dht

#### Usage

```go
node := dht.NewNode(dht.OptionAddress("0.0.0.0:8661"))
	node.PeerHandler = func(ip string, port int, hashInfo, peerID string) {
		log.Println("new announce_peer query", hashInfo)
	}
node.Serve()
```