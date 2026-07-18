package easyconnect

import (
	"crypto/tls"
	"testing"
)

func TestParseResourcesAcceptsServerWithoutRemoteDNS(t *testing.T) {
	client := NewClient("vpn.example.com:443", "", "", "", tls.Certificate{}, "", false, true, true)
	resources := `<Resource><Rcs><Rc type="1" proto="-1" host="10.1.2.3" port="1~65535"/></Rcs><Dns data="" dnsserver=""/></Resource>`

	if err := client.parseResources(resources); err != nil {
		t.Fatalf("parseResources() error = %v, want nil", err)
	}

	ipResources, err := client.IPResources()
	if err != nil {
		t.Fatalf("IPResources() error = %v", err)
	}
	if len(ipResources) != 1 {
		t.Fatalf("IPResources() = %#v, want one resource for 10.1.2.3", ipResources)
	}
	if got := ipResources[0].IPMin.String(); got != "10.1.2.3" {
		t.Fatalf("IPResources()[0].IPMin = %s, want 10.1.2.3", got)
	}

	if _, err := client.DNSServer(); err == nil {
		t.Fatal("DNSServer() error = nil, want unavailable remote DNS")
	}
}
