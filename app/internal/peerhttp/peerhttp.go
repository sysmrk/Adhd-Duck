// Package peerhttp secures controller/worker traffic. Configure once before
// constructing clients or starting goroutines. Registry and Kubernetes clients
// deliberately do not use this transport.
package peerhttp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

var configuration *tls.Config

// Configure loads a dedicated cluster CA and a certificate with clientAuth and
// serverAuth usages and the DNS SAN adhdduck.internal. Keys never leave Secrets.
func Configure(caFile, certFile, keyFile string) error {
	b, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("peer CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(b) {
		return fmt.Errorf("peer CA contains no certificates")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("peer certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("peer leaf: %w", err)
	}
	intermediates := x509.NewCertPool()
	for _, raw := range cert.Certificate[1:] {
		intermediate, err := x509.ParseCertificate(raw)
		if err != nil {
			return fmt.Errorf("peer chain: %w", err)
		}
		intermediates.AddCert(intermediate)
	}
	for _, usage := range []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth} {
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, Intermediates: intermediates, DNSName: "adhdduck.internal", KeyUsages: []x509.ExtKeyUsage{usage}}); err != nil {
			return fmt.Errorf("peer certificate verification: %w", err)
		}
	}
	configuration = &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, ClientCAs: pool, Certificates: []tls.Certificate{cert}, ServerName: "adhdduck.internal"}
	return nil
}

type transport struct{ base http.RoundTripper }

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	u := *r.URL
	clone.URL = &u
	// Peer addresses are discovered as IP:port. Verify the dedicated service SAN
	// instead of unstable pod IPs; never disable certificate verification.
	if configuration != nil {
		clone.URL.Scheme = "https"
	}
	return t.base.RoundTrip(clone)
}

// Client preserves operation-specific deadlines and dial settings, disables
// environment proxies and refuses redirects so credentials cannot escape peers.
func Client(c *http.Client) *http.Client {
	var t *http.Transport
	if c.Transport == nil {
		t = http.DefaultTransport.(*http.Transport).Clone()
	} else if original, ok := c.Transport.(*http.Transport); ok {
		t = original.Clone()
	} else {
		panic("peer client requires http.Transport")
	}
	t.Proxy = nil
	if configuration != nil {
		t.TLSClientConfig = configuration.Clone()
	}
	if t.ResponseHeaderTimeout == 0 {
		t.ResponseHeaderTimeout = 30 * time.Second
	}
	c.Transport = transport{t}
	c.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return fmt.Errorf("peer redirects are forbidden") }
	return c
}

// Serve uses optional client verification at handshake time so kubelet probes
// and the separately bearer-authenticated CI webhook can connect. Every other
// route requires a verified client certificate before reaching its handler.
func Serve(s *http.Server) error {
	if configuration == nil {
		return fmt.Errorf("peer TLS is not configured")
	}
	cfg := configuration.Clone()
	cfg.ClientAuth = tls.VerifyClientCertIfGiven
	cfg.ServerName = ""
	s.TLSConfig = cfg
	s.Handler = guard(s.Handler)
	return s.ListenAndServeTLS("", "")
}

func guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" && r.URL.Path != "/webhook/preheat" && (r.TLS == nil || len(r.TLS.VerifiedChains) == 0) {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
