package dht

type TokenManager struct {
}

var defaultTokenManager = new(TokenManager)

func (*TokenManager) GenToken() []byte {
	buf := make([]byte, 2)
	r.Read(buf)

	return buf
}
