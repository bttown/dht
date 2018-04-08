package dht

import (
	loglib "log"
	"os"
)

var log *loglib.Logger

func init() {
	log = loglib.New(os.Stderr, "[node] ", loglib.Lshortfile|loglib.Ltime)
}
