// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testtls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	material := Write(t)

	caCert := readCertificate(t, material.CACertPath)
	serverCert := readCertificate(t, material.ServerCertPath)
	clientCert := readCertificate(t, material.ClientCertPath)

	_, err := tls.LoadX509KeyPair(material.ServerCertPath, material.ServerKeyPath)
	require.NoError(t, err)

	_, err = tls.LoadX509KeyPair(material.ClientCertPath, material.ClientKeyPath)
	require.NoError(t, err)

	require.True(t, caCert.IsCA)
	require.Equal(t, "otelconf test ca", caCert.Subject.CommonName)
	require.Equal(t, "localhost", serverCert.Subject.CommonName)
	require.Equal(t, "otelconf test client", clientCert.Subject.CommonName)
	require.Equal(t, []string{"localhost"}, serverCert.DNSNames)
	require.Len(t, serverCert.IPAddresses, 2)
	require.True(t, serverCert.IPAddresses[0].Equal(net.IPv4(127, 0, 0, 1)))
	require.True(t, serverCert.IPAddresses[1].Equal(net.IPv6loopback))
	require.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, serverCert.ExtKeyUsage)
	require.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, clientCert.ExtKeyUsage)

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	_, err = serverCert.Verify(x509.VerifyOptions{
		DNSName: "localhost",
		Roots:   roots,
	})
	require.NoError(t, err)

	_, err = clientCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	require.NoError(t, err)
}

func readCertificate(t *testing.T, path string) *x509.Certificate {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	block, _ := pem.Decode(data)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}
