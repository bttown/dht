package dht

import (
	"fmt"
)

var (
	// KRPCErrGeneric ...
	KRPCErrGeneric = newKRPCError(201, "A Generic Error Ocurred")
	// KRPCErrServer ...
	KRPCErrServer = newKRPCError(201, "A Server Error Ocurred")
	// KPRCErrProtocol ...
	KPRCErrProtocol = newKRPCError(203, "A Protocol Error Ocurred")
	// KPRCErrMalformedPacket ...
	KPRCErrMalformedPacket = newKRPCError(203, "A Protocol Error Ocurred")
	// KRPCErrMethodUnknown ...
	KRPCErrMethodUnknown = newKRPCError(204, "Method Unknown")
)

type krpcError struct {
	Code        int
	Description string
	s           string
}

func (err *krpcError) Error() string {
	return err.s
}

func newKRPCError(code int, desc string) error {
	return &krpcError{Code: code, Description: desc, s: fmt.Sprintf("<%d>%s", code, desc)}
}
