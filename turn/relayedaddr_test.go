package turn

import (
	"net"
	"testing"

	"github.com/nkbai/goice/stun"
)

func TestRelayedAddress(t *testing.T) {
	// Simple tests because already tested in stun.
	a := RelayedAddress{
		IP:   net.IPv4(111, 11, 1, 2),
		Port: 333,
	}
	m := new(stun.Message)
	if err := a.AddTo(m); err != nil {
		t.Fatal(err)
	}
	m.WriteHeader()
	decoded := new(stun.Message)
	decoded.Write(m.Raw)
	var aGot RelayedAddress
	if err := aGot.GetFrom(decoded); err != nil {
		t.Fatal(err)
	}
}
