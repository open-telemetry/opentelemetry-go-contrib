// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package testtls provides runtime-generated TLS materials for tests.
package testtls // import "go.opentelemetry.io/contrib/otelconf/internal/testtls"

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Material contains generated CA, server, and client cert file paths.
type Material struct {
	CACertPath     string
	CAKeyPath      string
	ServerCertPath string
	ServerKeyPath  string
	ClientCertPath string
	ClientKeyPath  string
}

// TB is the minimal testing surface needed by this helper.
type TB interface {
	Helper()
	TempDir() string
	Fatalf(format string, args ...any)
}

// Write generates mTLS assets under t.TempDir so tests do not depend on expiring fixtures.
func Write(t TB) Material {
	t.Helper()

	dir := t.TempDir()
	now := time.Now().UTC()

	caKey := mustRSAKey(t)
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
			Organization: []string{"OpenTelemetry Test CA"},
			CommonName:   "otelconf test ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER := mustCertificate(t, caTemplate, caTemplate, &caKey.PublicKey, caKey)

	serverKey := mustRSAKey(t)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
			Organization: []string{"OpenTelemetry Tests"},
			CommonName:   "localhost",
		},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(2, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	serverDER := mustCertificate(t, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)

	clientKey := mustRSAKey(t)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano() + 1),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
			Organization: []string{"OpenTelemetry Tests"},
			CommonName:   "otelconf test client",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(2, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	clientDER := mustCertificate(t, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)

	m := Material{
		CACertPath:     filepath.Join(dir, "ca.crt"),
		CAKeyPath:      filepath.Join(dir, "ca.key"),
		ServerCertPath: filepath.Join(dir, "server.crt"),
		ServerKeyPath:  filepath.Join(dir, "server.key"),
		ClientCertPath: filepath.Join(dir, "client.crt"),
		ClientKeyPath:  filepath.Join(dir, "client.key"),
	}
	writeCert(t, m.CACertPath, caDER)
	writeKey(t, m.CAKeyPath, caKey)
	writeCert(t, m.ServerCertPath, serverDER)
	writeKey(t, m.ServerKeyPath, serverKey)
	writeCert(t, m.ClientCertPath, clientDER)
	writeKey(t, m.ClientKeyPath, clientKey)
	return m
}

func mustRSAKey(t TB) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	return key
}

func mustCertificate(t TB, template, parent *x509.Certificate, publicKey *rsa.PublicKey, signer *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, signer)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return der
}

func writeCert(t TB, path string, der []byte) {
	t.Helper()
	block := &pem.Block{Type: "CERTIFICATE", Bytes: der}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write cert %s: %v", path, err)
	}
}

func writeKey(t TB, path string, key *rsa.PrivateKey) {
	t.Helper()
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write key %s: %v", path, err)
	}
}
