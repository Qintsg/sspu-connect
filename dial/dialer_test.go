package dial

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/mythologyli/zju-connect/client"
	"github.com/mythologyli/zju-connect/internal/ippool"
	"github.com/mythologyli/zju-connect/internal/zcdns"
	"github.com/mythologyli/zju-connect/resolve"
)

type recordingStack struct {
	dialCount int
}

func (*recordingStack) Run()                                              {}
func (*recordingStack) SetupResolve(zcdns.LocalServer)                    {}
func (*recordingStack) SetupIPPool(*ippool.IPPool[client.DomainResource]) {}
func (s *recordingStack) DialTCP(context.Context, *net.TCPAddr) (net.Conn, error) {
	s.dialCount++
	return nil, errors.New("unexpected VPN dial")
}
func (s *recordingStack) DialUDP(context.Context, *net.UDPAddr) (net.Conn, error) {
	s.dialCount++
	return nil, errors.New("unexpected VPN dial")
}

func TestDialIPPortRejectsClashFakeIPBeforeVPN(t *testing.T) {
	stack := &recordingStack{}
	dialer := &Dialer{stack: stack}
	resource := client.DomainResource{PortMin: 1, PortMax: 65535, Protocol: "all"}
	ctx := context.WithValue(context.Background(), resolve.ContextKeyDomainResource, resource)
	ctx = context.WithValue(ctx, resolve.ContextKeyResolveHost, "library.sspu.edu.cn")

	_, err := dialer.DialIPPort(ctx, "tcp", "198.18.1.168:443")
	if !errors.Is(err, ErrFakeIPDestination) {
		t.Fatalf("DialIPPort() error = %v, want ErrFakeIPDestination", err)
	}
	if stack.dialCount != 0 {
		t.Fatalf("VPN dial count = %d, want 0", stack.dialCount)
	}
}
