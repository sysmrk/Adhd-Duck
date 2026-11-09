package peerhttp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMutualTLSAndRouteBoundaries(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(2), DNSNames: []string{"adhdduck.internal"}, NotBefore: ca.NotBefore, NotAfter: ca.NotAfter, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	dir := t.TempDir()
	for name, b := range map[string][]byte{"ca.crt": pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), "tls.crt": pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), "tls.key": pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Configure(filepath.Join(dir, "ca.crt"), filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")); err != nil {
		t.Fatal(err)
	}
	defer func() { configuration = nil }()
	srv := httptest.NewUnstartedServer(guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "https://example.invalid", 302)
			return
		}
		w.WriteHeader(204)
	})))
	srv.TLS = configuration.Clone()
	srv.TLS.ClientAuth = tls.VerifyClientCertIfGiven
	srv.StartTLS()
	defer srv.Close()
	client := Client(&http.Client{Timeout: time.Second})
	// Existing http discovery URLs must upgrade to TLS, never send a bearer in clear.
	resp, err := client.Get("http://" + srv.Listener.Addr().String() + "/report")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatal(resp.StatusCode)
	}
	if _, err := client.Get(srv.URL + "/redirect"); err == nil {
		t.Fatal("redirect followed")
	}
	noCert := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: configuration.RootCAs, ServerName: "adhdduck.internal", MinVersion: tls.VersionTLS13}}}
	for path, want := range map[string]int{"/report": 401, "/status": 401, "/metrics": 401, "/healthz": 204, "/webhook/preheat": 204} {
		resp, err := noCert.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
	}
	badCA := Client(&http.Client{Timeout: time.Second})
	tr := badCA.Transport.(transport)
	tr.base.(*http.Transport).TLSClientConfig.RootCAs = x509.NewCertPool()
	if _, err := badCA.Get(srv.URL + "/report"); err == nil {
		t.Fatal("untrusted server accepted")
	}
}
