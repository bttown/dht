package dht

import (
	loglib "log"
	"os"
)

var log *loglib.Logger

func init() {
	log = loglib.New(os.Stderr, "[collector] ", loglib.Lshortfile|loglib.Ltime)
}
