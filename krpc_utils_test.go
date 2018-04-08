package dht

import (
	"fmt"
	"testing"
)

func TestKRPPingQueryEncode(t *testing.T) {
	query := KRPCQuery{
		T:         "aa",
		Q:         FindNodeType,
		TargetNID: GenerateNodeID(20),
	}

	out, err := query.Encode()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(out))
}
