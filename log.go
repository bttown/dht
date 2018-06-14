package dht

import (
	loglib "log"
	"os"
)

var log *loglib.Logger

func init() {
	log = loglib.New(os.Stderr, "[dht-node] ", loglib.Lshortfile|loglib.Ltime)
}
