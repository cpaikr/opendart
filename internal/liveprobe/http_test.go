package liveprobe

import (
	"crypto/tls"
	"net/http"
	"slices"
	"testing"
	"time"
)

func TestNewSequentialHTTPClientConfinesTransportPolicy(t *testing.T) {
	client := NewSequentialHTTPClient(17 * time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T", client.Transport)
	}
	if client.Timeout != 17*time.Second || transport.Proxy != nil || !transport.DisableKeepAlives || transport.ForceAttemptHTTP2 || len(transport.TLSNextProto) != 0 {
		t.Fatalf("client transport policy was not applied: %#v", transport)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLS config = %#v", transport.TLSClientConfig)
	}
	if !slices.Contains(transport.TLSClientConfig.CipherSuites, tls.TLS_RSA_WITH_AES_128_GCM_SHA256) {
		t.Fatal("required OpenDART compatibility cipher is absent")
	}
	if len(transport.TLSClientConfig.CipherSuites) <= 1 || transport.TLSClientConfig.CipherSuites[0] == tls.TLS_RSA_WITH_AES_128_GCM_SHA256 {
		t.Fatal("compatibility cipher displaced Go's secure suite set")
	}
	if err := client.CheckRedirect(&http.Request{}, nil); err == nil {
		t.Fatal("redirect policy accepted a redirect")
	}
}
