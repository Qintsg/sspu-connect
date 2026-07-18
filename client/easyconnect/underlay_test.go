package easyconnect

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSetupUnderlayUsesManualInterfaceForHTTP(t *testing.T) {
	client := NewClient("vpn.example.com:443", "", "", "", tls.Certificate{}, "", false, false, false)
	client.setupUnderlay("manual-interface", true)

	if got := client.underlayDialer.InterfaceName(); got != "manual-interface" {
		t.Fatalf("underlay interface = %q, want %q", got, "manual-interface")
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("HTTP transport has type %T, want *http.Transport", client.httpClient.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("HTTP transport does not use the underlay dialer")
	}
}

func TestCertificateTransportKeepsUnderlayDialer(t *testing.T) {
	client := NewClient("vpn.example.com:443", "", "", "", tls.Certificate{}, "", false, false, false)
	client.setupUnderlay("", false)
	client.setHTTPTransport(&tls.Config{Renegotiation: tls.RenegotiateOnceAsClient})

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("HTTP transport has type %T, want *http.Transport", client.httpClient.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("certificate HTTP transport does not use the underlay dialer")
	}
	if transport.TLSClientConfig.Renegotiation != tls.RenegotiateOnceAsClient {
		t.Fatal("certificate TLS configuration was not preserved")
	}
}

func TestSessionKeepAliveSendsRequestBeforeCancelingContext(t *testing.T) {
	requestReceived := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/por/logout.csp" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/por/update_session.csp" {
			t.Errorf("request path = %q, want /por/update_session.csp", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		requestReceived <- struct{}{}
		_, _ = w.Write([]byte(`<Auth><Message>success</Message><ErrorCode>1</ErrorCode></Auth>`))
	}))
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "https://"), "", "", "", tls.Certificate{}, "session-id", false, false, false)
	client.httpClient = server.Client()
	originalInterval := sessionKeepAliveInterval
	sessionKeepAliveInterval = time.Millisecond
	t.Cleanup(func() {
		sessionKeepAliveInterval = originalInterval
		client.Close()
	})

	go client.sessionKeepAliveLoop()
	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("session keepalive request was not received")
	}
}

func TestCloseLogsOutAuthorizedSession(t *testing.T) {
	requestReceived := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/por/logout.csp" {
			t.Errorf("request path = %q, want /por/logout.csp", r.URL.Path)
		}
		if cookie, err := r.Cookie("TWFID"); err != nil || cookie.Value != "session-id" {
			t.Errorf("TWFID cookie = %v, %v; want session-id", cookie, err)
		}
		requestReceived <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "https://"), "", "", "", tls.Certificate{}, "session-id", false, false, false)
	client.httpClient = server.Client()
	client.Close()

	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("logout request was not received")
	}
}
