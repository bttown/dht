package dht

import (
	"math/rand"
	"time"
)

type TokenManager struct {
}

var defaultTokenManager = new(TokenManager)

func (*TokenManager) GenToken() []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 2)
	r.Read(buf)

	return buf
}
