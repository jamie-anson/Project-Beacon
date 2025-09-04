package serverbind

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

// ResolveAndListen applies the binding strategy and returns a bound listener and its resolved address.
// - strategy: strict | fallback | ephemeral
// - desired: the configured address (e.g., ":8090")
// - rangeStart/rangeEnd: inclusive scan range used by fallback when desired is default :8090 and busy
func ResolveAndListen(strategy, desired string, rangeStart, rangeEnd int) (net.Listener, string, error) {
	strategy = strings.ToLower(strings.TrimSpace(strategy))

	addrToTry := desired
	if strategy == "ephemeral" {
		addrToTry = ":0"
	}

	// Strict: single attempt
	if strategy == "strict" {
		ln, err := net.Listen("tcp", addrToTry)
		if err != nil {
			return nil, "", err
		}
		return ln, ln.Addr().String(), nil
	}

	// Ephemeral: bind :0
	if strategy == "ephemeral" {
		ln, err := net.Listen("tcp", addrToTry)
		if err != nil {
			return nil, "", err
		}
		return ln, ln.Addr().String(), nil
	}

	// Fallback: try desired, then scan range if default :8090 busy
	ln, err := net.Listen("tcp", addrToTry)
	if err == nil {
		return ln, ln.Addr().String(), nil
	}

	if desired == ":8090" && (strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "bind")) {
		for p := rangeStart; p <= rangeEnd; p++ {
			cand := ":" + strconv.Itoa(p)
			ln2, err2 := net.Listen("tcp", cand)
			if err2 == nil {
				return ln2, ln2.Addr().String(), nil
			}
		}
		return nil, "", errors.New("exhausted fallback port range")
	}

	return nil, "", err
}

// WriteAddrFile writes the resolved address to the specified path.
func WriteAddrFile(path, addr string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return osWriteFile(path, []byte(addr))
}

// osWriteFile is isolated for unit testability
var osWriteFile = func(path string, b []byte) error {
	return osWriteFileReal(path, b)
}

var osWriteFileReal = func(path string, b []byte) error {
	return writeFile(path, b)
}

// writeFile is defined in a tiny platform shim in the same package
