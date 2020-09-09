// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortex

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthentication checks whether http requests are properly authenticated with either
// bearer tokens or basic authentication in the addHeaders method.
func TestAuthentication(t *testing.T) {
	tests := []struct {
		testName                      string
		basicAuth                     map[string]string
		basicAuthPasswordFileContents []byte
		bearerToken                   string
		bearerTokenFile               string
		bearerTokenFileContents       []byte
		expectedAuthHeaderValue       string
		expectedError                 error
	}{
		{
			testName: "Basic Auth with password",
			basicAuth: map[string]string{
				"username": "TestUser",
				"password": "TestPassword",
			},
			expectedAuthHeaderValue: "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("TestUser:TestPassword"),
			),
			expectedError: nil,
		},
		{
			testName: "Basic Auth with password file",
			basicAuth: map[string]string{
				"username":      "TestUser",
				"password_file": "passwordFile",
			},
			basicAuthPasswordFileContents: []byte("TestPassword"),
			expectedAuthHeaderValue: "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("TestUser:TestPassword"),
			),
			expectedError: nil,
		},
		{
			testName: "Basic Auth with bad password file",
			basicAuth: map[string]string{
				"username":      "TestUser",
				"password_file": "missingPasswordFile",
			},
			expectedAuthHeaderValue: "",
			expectedError:           ErrFailedToReadFile,
		},
		{
			testName:                "Bearer Token",
			bearerToken:             "testToken",
			expectedAuthHeaderValue: "Bearer testToken",
			expectedError:           nil,
		},
		{
			testName:                "Bearer Token with bad bearer token file",
			bearerTokenFile:         "missingBearerTokenFile",
			expectedAuthHeaderValue: "",
			expectedError:           ErrFailedToReadFile,
		},
		{
			testName:                "Bearer Token with bearer token file",
			bearerTokenFile:         "bearerTokenFile",
			expectedAuthHeaderValue: "Bearer testToken",
			bearerTokenFileContents: []byte("testToken"),
			expectedError:           nil,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Set up a test server that runs a handler function when it receives a HTTP
			// request. The server writes the request's Authorization header to the
			// response body.
			handler := func(rw http.ResponseWriter, req *http.Request) {
				authHeaderValue := req.Header.Get("Authorization")
				_, err := rw.Write([]byte(authHeaderValue))
				require.NoError(t, err)
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// Create the necessary files for tests.
			if test.basicAuth != nil {
				passwordFile := test.basicAuth["password_file"]
				if passwordFile != "" && test.basicAuthPasswordFileContents != nil {
					filepath := "./" + test.basicAuth["password_file"]
					err := createFile(test.basicAuthPasswordFileContents, filepath)
					require.NoError(t, err)
					defer os.Remove(filepath)
				}
			}
			if test.bearerTokenFile != "" && test.bearerTokenFileContents != nil {
				filepath := "./" + test.bearerTokenFile
				err := createFile(test.bearerTokenFileContents, filepath)
				require.NoError(t, err)
				defer os.Remove(filepath)
			}

			// Create a HTTP request and add headers to it through an Exporter. Since the
			// Exporter has an empty Headers map, authentication methods will be called.
			exporter := Exporter{
				Config{
					BasicAuth:       test.basicAuth,
					BearerToken:     test.bearerToken,
					BearerTokenFile: test.bearerTokenFile,
				},
			}
			req, err := http.NewRequest(http.MethodPost, server.URL, nil)
			require.NoError(t, err)
			err = exporter.addHeaders(req)

			// Verify the error and if the Authorization header was correctly set.
			if err != nil {
				require.Equal(t, err.Error(), test.expectedError.Error())
			} else {
				require.NoError(t, test.expectedError)
				authHeaderValue := req.Header.Get("Authorization")
				require.Equal(t, authHeaderValue, test.expectedAuthHeaderValue)
			}
		})
	}
}

// createFile writes a file with a slice of bytes at a specified filepath.
func createFile(bytes []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// TestBuildClient checks whether the buildClient successfully creates a client that can
// connect over TLS and has the correct remote timeout and proxy url.
func TestBuildClient(t *testing.T) {
	testProxyURL, err := url.Parse("123.4.5.6")
	require.NoError(t, err)

	tests := []struct {
		testName               string
		config                 Config
		expectedRemoteTimeout  time.Duration
		expectedErrorSubstring string
	}{
		{
			testName: "Remote Timeout with Proxy URL",
			config: Config{
				ProxyURL:      testProxyURL,
				RemoteTimeout: 123 * time.Second,
				TLSConfig: map[string]string{
					"ca_file":              "./ca_cert.pem",
					"insecure_skip_verify": "0",
				},
			},
			expectedRemoteTimeout:  123 * time.Second,
			expectedErrorSubstring: "proxyconnect tcp",
		},
		{
			testName: "No Timeout or Proxy URL, InsecureSkipVerify is false",
			config: Config{
				TLSConfig: map[string]string{
					"ca_file":              "./ca_cert.pem",
					"insecure_skip_verify": "0",
				},
			},
			expectedErrorSubstring: "",
		},
		{
			testName: "No Timeout or Proxy URL, InsecureSkipVerify is true",
			config: Config{
				TLSConfig: map[string]string{
					"ca_file":              "./ca_cert.pem",
					"insecure_skip_verify": "1",
				},
			},
			expectedErrorSubstring: "",
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Create and start the TLS server.
			handler := func(rw http.ResponseWriter, req *http.Request) {
				fmt.Fprint(rw, "Successfully received HTTP request!")
			}
			server := httptest.NewTLSServer(http.HandlerFunc(handler))
			defer server.Close()

			// Create a certicate for the CA from the TLS server. This will be used to
			// verify the test server by the client.
			encodedCACert := server.TLS.Certificates[0].Certificate[0]
			caCertPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: encodedCACert,
			})
			err := createFile(caCertPEM, "./ca_cert.pem")
			require.NoError(t, err)
			defer os.Remove("ca_cert.pem")

			// Create an Exporter client and check the timeout.
			exporter := Exporter{
				config: test.config,
			}
			client, err := exporter.buildClient()
			require.NoError(t, err)
			assert.Equal(t, client.Timeout, test.expectedRemoteTimeout)

			// Attempt to send the request and verify that the correct error occurred. If
			// an error is expected, the test checks if the error string contains the
			// expected error substring since the error can contain the server URL, which
			// changes every test.
			_, err = client.Get(server.URL)
			if test.expectedErrorSubstring != "" {
				if assert.Error(t, err) {
					hasErrorSubstring := strings.Contains(err.Error(), test.expectedErrorSubstring)
					assert.True(t, hasErrorSubstring, "missing error message")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMutualTLS is an integration test that checks whether the Exporter's client can
// successfully verify a server and send a HTTP request and whether a server can
// successfully verify the Exporter client and receive the HTTP request.
func TestMutualTLS(t *testing.T) {
	tests := []struct {
		testName      string
		generateCerts bool
		caCert        string
		caKey         string
		servingCert   string
		servingKey    string
		clientCert    string
		clientKey     string
	}{
		{
			testName:      "Generated ECDSA Certs",
			generateCerts: true,
			caCert:        "ca.crt",
			caKey:         "ca.key",
			servingCert:   "server.crt",
			servingKey:    "server.key",
			clientCert:    "client.crt",
			clientKey:     "client.key",
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			if test.generateCerts {
				// Generate certificate authority certificate to sign other certificates.
				caCert, caPrivateKey, err := generateCACertFiles(test.caCert, test.caKey)
				require.NoError(t, err)
				defer os.Remove(test.caCert)
				defer os.Remove(test.caKey)

				// Generate certificate for the server. The client will check this
				// certificate against its certificate authority to verify the server.
				_, _, err = generateServingCertFiles(
					caCert,
					caPrivateKey,
					test.servingCert,
					test.servingKey,
				)
				require.NoError(t, err)
				defer os.Remove(test.servingCert)
				defer os.Remove(test.servingKey)

				// Generate certificate for the client. The server will check this
				// certificate against its certificate authority to verify the client.
				_, _, err = generateClientCertFiles(
					caCert,
					caPrivateKey,
					test.clientCert,
					test.clientKey,
				)
				require.NoError(t, err)
				defer os.Remove(test.clientCert)
				defer os.Remove(test.clientKey)
			}

			// Generate the TLS Config to set up mutual TLS on the server.
			serverTLSConfig, err := generateServerTLSConfig(
				test.caCert,
				test.servingCert,
				test.servingKey,
			)
			require.NoError(t, err)

			// Create and start the TLS server.
			handler := func(rw http.ResponseWriter, req *http.Request) {
				fmt.Fprint(rw, "Successfully verified client and received request!")
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(handler))
			server.TLS = serverTLSConfig
			server.StartTLS()
			defer server.Close()

			// Create an Exporter client with the client and CA certificate files.
			exporter := Exporter{
				Config{
					TLSConfig: map[string]string{
						"ca_file":              test.caCert,
						"cert_file":            test.clientCert,
						"key_file":             test.clientKey,
						"insecure_skip_verify": "0",
					},
				},
			}
			client, err := exporter.buildClient()
			require.NoError(t, err)

			// Send the request and verify that the request was successfully received.
			res, err := client.Get(server.URL)
			require.NoError(t, err)
			defer res.Body.Close()
		})
	}
}

// generateCertFiles generates new certificate files from a template that is signed with
// the provided signer certificate and key.
func generateCertFiles(
	template *x509.Certificate,
	signer *x509.Certificate,
	signerKey *ecdsa.PrivateKey,
	certFilepath string,
	keyFilepath string,
) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	// Generate a private key for the new certificate. This does not have to be rsa 4096.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Check if a signer key was provided. If not, then use the newly created key.
	if signerKey == nil {
		signerKey = privateKey
	}

	// Create a new certificate in DER encoding.
	encodedCert, err := x509.CreateCertificate(
		rand.Reader, template, signer, privateKey.Public(), signerKey,
	)
	if err != nil {
		return nil, nil, err
	}

	// Write the certificate to the local directory.
	certFile, err := os.Create(certFilepath)
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: encodedCert,
	})
	if err != nil {
		return nil, nil, err
	}

	// Write the private key to the local directory.
	encodedPrivateKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	keyFile, err := os.Create(keyFilepath)
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: encodedPrivateKey,
	})
	if err != nil {
		return nil, nil, err
	}

	// Parse the newly created certificate so it can be returned.
	cert, err := x509.ParseCertificate(encodedCert)
	if err != nil {
		return nil, nil, err
	}
	return cert, privateKey, nil
}

// generateCACertFiles creates a CA certificate and key in the local directory. This
// certificate is used to sign other certificates.
func generateCACertFiles(certFilepath string, keyFilepath string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	// Create a template for CA certificates.
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(123),
		Subject: pkix.Name{
			Organization: []string{"CA Certificate"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(5 * time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	// Create the certificate files. CA certificates are used to sign other certificates
	// so it signs itself with its own template and private key during creation.
	cert, privateKey, err := generateCertFiles(
		certTemplate,
		certTemplate,
		nil,
		certFilepath,
		keyFilepath,
	)
	if err != nil {
		return nil, nil, err
	}

	return cert, privateKey, nil
}

// generateServingCertFiles creates a new certificate that a client will check against its
// certificate authority to verify the server. The certificate is signed by a certificate
// authority.
func generateServingCertFiles(
	caCert *x509.Certificate,
	caPrivateKey *ecdsa.PrivateKey,
	certFilepath string,
	keyFilepath string,
) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(456),
		Subject: pkix.Name{
			Organization: []string{"Serving Certificate"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(5 * time.Minute),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create the certificate files. The CA certificate is used to sign the new
	// certificate.
	cert, privateKey, err := generateCertFiles(
		certTemplate,
		caCert,
		caPrivateKey,
		certFilepath,
		keyFilepath,
	)
	if err != nil {
		return nil, nil, err
	}

	return cert, privateKey, nil
}

// generateClientCertFiles creates a new certificate that a server will check against its
// certificate authority to verify the client. The certificate is signed by a certificate
// authority.
func generateClientCertFiles(
	caCert *x509.Certificate,
	caPrivateKey *ecdsa.PrivateKey,
	certFilepath string,
	keyFilepath string,
) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(789),
		Subject: pkix.Name{
			Organization: []string{"Client Certificate"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(5 * time.Minute),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create the certificate files. The CA certificate is used to sign the new
	// certificate.
	cert, privateKey, err := generateCertFiles(
		certTemplate,
		caCert,
		caPrivateKey,
		certFilepath,
		keyFilepath,
	)
	if err != nil {
		return nil, nil, err
	}

	return cert, privateKey, nil
}

// generateServerTLSConfig creates a tls Config struct for a server that wants to both
// verify servers and have clients verify itself.
func generateServerTLSConfig(caCertFile string, servingCertFile string, servingKeyFile string) (*tls.Config, error) {
	// Create the server's serving certificate. This allows clients to verify the server.
	servingCert, err := tls.LoadX509KeyPair(servingCertFile, servingKeyFile)
	if err != nil {
		return nil, err
	}

	// Create a certificate pool to store the CA certificate. This allows the server to
	// verify client certificates signed by the stored certicate authority.
	encodedCACert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(encodedCACert)

	// Create the tls Config struct and set it to always verify the client with the CAs.
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{servingCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	return tlsConfig, nil
}
