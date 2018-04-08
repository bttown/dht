package dht

import (
	"testing"
)

func TestGenerateNodeID(t *testing.T) {
	length := 20
	id := GenerateNodeID(length)
	if len(id) != length {
		t.Errorf("a id %d length long was expected, but go %d length long", length, len(id))
	}
}
