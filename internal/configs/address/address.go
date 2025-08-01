package address

import (
	"errors"
	"strings"
)

// Supported scheme constants.
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
	SchemeGRPC  = "grpc"
)

// ErrUnsupportedScheme is returned when an address uses an unknown or unsupported scheme.
var ErrUnsupportedScheme = errors.New("unsupported address scheme")

// Address holds the scheme and actual network address.
type Address struct {
	Scheme  string
	Address string
}

// New parses the full input address and returns separated scheme/address.
// Default scheme is "http" if not specified.
func New(input string) Address {
	scheme := SchemeHTTP
	addr := input

	if strings.HasPrefix(addr, SchemeHTTP+"://") {
		scheme = SchemeHTTP
		addr = strings.TrimPrefix(addr, SchemeHTTP+"://")
	} else if strings.HasPrefix(addr, SchemeHTTPS+"://") {
		scheme = SchemeHTTPS
		addr = strings.TrimPrefix(addr, SchemeHTTPS+"://")
	} else if strings.HasPrefix(addr, SchemeGRPC+"://") {
		scheme = SchemeGRPC
		addr = strings.TrimPrefix(addr, SchemeGRPC+"://")
	}

	return Address{
		Scheme:  scheme,
		Address: addr,
	}
}
