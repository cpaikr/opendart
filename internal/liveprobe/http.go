// Package liveprobe owns transport invariants shared by explicitly invoked,
// credentialed OpenDART observations.
package liveprobe

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"
)

// NewSequentialHTTPClient returns a one-attempt client with no redirects,
// connection reuse, or HTTP/2 negotiation. The upstream currently requires a
// TLS 1.2 RSA key-exchange suite, so that single compatibility suite is added
// to Go's currently secure suite set and confined to live observation tooling.
func NewSequentialHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	transport.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: probeCipherSuites(),
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirects are not allowed")
		},
	}
}

func probeCipherSuites() []uint16 {
	suites := tls.CipherSuites()
	result := make([]uint16, 0, len(suites)+1)
	for _, suite := range suites {
		result = append(result, suite.ID)
	}
	return append(result, tls.TLS_RSA_WITH_AES_128_GCM_SHA256)
}
