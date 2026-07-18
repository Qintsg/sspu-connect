package resolve

import (
	"context"
	"net"
	"testing"

	"github.com/mythologyli/zju-connect/client"
)

func TestResolveDoesNotMatchDomainWithoutLabelBoundary(t *testing.T) {
	resource := client.DomainResource{PortMin: 1, PortMax: 65535, Protocol: "all"}
	resolver := NewResolver(
		nil,
		"",
		"",
		60,
		map[string]client.DomainResource{"sspu.edu.cn": resource},
		map[string]net.IP{"notsspu.edu.cn": net.ParseIP("203.0.113.10")},
		false,
	)

	ctx, _, err := resolver.Resolve(context.Background(), "notsspu.edu.cn")
	if err != nil {
		t.Fatal(err)
	}
	if got := ctx.Value(ContextKeyDomainResource); got != nil {
		t.Fatalf("unrelated domain matched SSPU rule: %#v", got)
	}
}

func TestDomainMatchesExactAndSubdomainRules(t *testing.T) {
	for _, host := range []string{"sspu.edu.cn", "library.sspu.edu.cn", "LIBRARY.SSPU.EDU.CN."} {
		if !domainMatches(host, "*.sspu.edu.cn") {
			t.Errorf("domainMatches(%q, %q) = false, want true", host, "*.sspu.edu.cn")
		}
	}
	for _, rule := range []string{"", "*", "."} {
		if domainMatches("www.example.com", rule) {
			t.Errorf("domainMatches(%q, %q) = true, want false", "www.example.com", rule)
		}
	}
}
