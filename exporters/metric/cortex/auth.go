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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

// ErrFailedToReadFile occurs when a password / bearer token file exists, but could
// not be read.
var ErrFailedToReadFile = fmt.Errorf("failed to read password / bearer token file")

// addBasicAuth sets the Authorization header for basic authentication using a username
// and a password / password file. The header value is not changed if an Authorization
// header already exists and no action is taken if the Exporter is not configured with
// basic authorization credentials.
func (e *Exporter) addBasicAuth(req *http.Request) error {
	// No need to add basic auth if it isn't provided or if the Authorization header is
	// already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}
	if e.config.BasicAuth == nil {
		return nil
	}

	username := e.config.BasicAuth["username"]

	// Use password from password file if it exists.
	passwordFile := e.config.BasicAuth["password_file"]
	if passwordFile != "" {
		file, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		password := string(file)
		req.SetBasicAuth(username, password)
		return nil
	}

	// Use provided password.
	password := e.config.BasicAuth["password"]
	req.SetBasicAuth(username, password)

	return nil
}

// addBearerTokenAuth sets the Authorization header for bearer tokens using a bearer token
// string or a bearer token file. The header value is not changed if an Authorization
// header already exists and no action is taken if the Exporter is not configured with
// bearer token credentials.
func (e *Exporter) addBearerTokenAuth(req *http.Request) error {
	// No need to add bearer token auth if the Authorization header is already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}

	// Use bearer token from bearer token file if it exists.
	if e.config.BearerTokenFile != "" {
		file, err := ioutil.ReadFile(e.config.BearerTokenFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		bearerTokenString := "Bearer " + string(file)
		req.Header.Set("Authorization", bearerTokenString)
		return nil
	}

	// Otherwise, use bearer token field.
	if e.config.BearerToken != "" {
		bearerTokenString := "Bearer " + e.config.BearerToken
		req.Header.Set("Authorization", bearerTokenString)
	}

	return nil
}

// buildClient returns a http client that uses TLS and has the user-specified proxy and
// timeout.
func (e *Exporter) buildClient() (*http.Client, error) {
	// Create a TLS Config struct for use in a custom HTTP Transport.
	tlsConfig, err := e.buildTLSConfig()
	if err != nil {
		return nil, err
	}

	// Create a custom HTTP Transport for the client. This is the same as
	// http.DefaultTransport other than the TLSClientConfig.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	// Convert proxy url to proxy function for use in the created Transport.
	if e.config.ProxyURL != nil {
		proxy := http.ProxyURL(e.config.ProxyURL)
		transport.Proxy = proxy
	}

	client := http.Client{
		Transport: transport,
		Timeout:   e.config.RemoteTimeout,
	}
	return &client, nil
}

// buildTLSConfig creates a new TLS Config struct with the properties from the exporter's
// Config struct.
func (e *Exporter) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if e.config.TLSConfig == nil {
		return tlsConfig, nil
	}

	// Set the server name if it exists.
	if e.config.TLSConfig["server_name"] != "" {
		tlsConfig.ServerName = e.config.TLSConfig["server_name"]
	}

	// Set InsecureSkipVerify. Viper reads the bool as a string since it is in a map.
	if isv, ok := e.config.TLSConfig["insecure_skip_verify"]; ok {
		var err error
		if tlsConfig.InsecureSkipVerify, err = strconv.ParseBool(isv); err != nil {
			return nil, err
		}
	}

	// Load certificates from CA file if it exists.
	caFile := e.config.TLSConfig["ca_file"]
	if caFile != "" {
		caFileData, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caFileData)
		tlsConfig.RootCAs = certPool
	}

	// Load the client certificate if it exists.
	certFile := e.config.TLSConfig["cert_file"]
	keyFile := e.config.TLSConfig["key_file"]
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
