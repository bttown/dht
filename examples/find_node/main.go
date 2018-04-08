package main

import (
	"github.com/bttown/dht"
)

func main() {
	node := dht.NewNode(dht.OptionAddress("0.0.0.0:8661"))
	node.Serve()
}
