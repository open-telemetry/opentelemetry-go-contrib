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

package semconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredHTTPPort(t *testing.T) {
	tests := []struct {
		https bool
		port  int
		want  int
	}{
		{true, 443, -1},
		{true, 80, 80},
		{true, 8081, 8081},
		{false, 443, 443},
		{false, 80, -1},
		{false, 8080, 8080},
	}
	for _, test := range tests {
		got := requiredHTTPPort(test.https, test.port)
		assert.Equal(t, test.want, got, test.https, test.port)
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		hostport string
		host     string
		port     int
	}{
		{"", "", -1},
		{":8080", "", 8080},
		{"127.0.0.1", "127.0.0.1", -1},
		{"www.example.com", "www.example.com", -1},
		{"127.0.0.1%25en0", "127.0.0.1%25en0", -1},
		{"[]", "", -1}, // Ensure this doesn't panic.
		{"[fe80::1", "", -1},
		{"[fe80::1]", "fe80::1", -1},
		{"[fe80::1%25en0]", "fe80::1%25en0", -1},
		{"[fe80::1]:8080", "fe80::1", 8080},
		{"[fe80::1]::", "", -1}, // Too many colons.
		{"127.0.0.1:", "127.0.0.1", -1},
		{"127.0.0.1:port", "127.0.0.1", -1},
		{"127.0.0.1:8080", "127.0.0.1", 8080},
		{"www.example.com:8080", "www.example.com", 8080},
		{"127.0.0.1%25en0:8080", "127.0.0.1%25en0", 8080},
	}

	for _, test := range tests {
		h, p := splitHostPort(test.hostport)
		assert.Equal(t, test.host, h, test.hostport)
		assert.Equal(t, test.port, p, test.hostport)
	}
}

func TestHTTPServerClientIP(t *testing.T) {
	tests := []struct {
		xForwardedFor string
		want          string
	}{
		{"", ""},
		{"127.0.0.1", "127.0.0.1"},
		{"127.0.0.1,127.0.0.5", "127.0.0.1"},
	}
	for _, test := range tests {
		got := serverClientIP(test.xForwardedFor)
		assert.Equal(t, test.want, got, test.xForwardedFor)
	}
}

func TestNetProtocol(t *testing.T) {
	type testCase struct {
		name, version string
	}
	tests := map[string]testCase{
		"HTTP/1.0":        {name: "http", version: "1.0"},
		"HTTP/1.1":        {name: "http", version: "1.1"},
		"HTTP/2":          {name: "http", version: "2"},
		"HTTP/3":          {name: "http", version: "3"},
		"SPDY":            {name: "spdy"},
		"SPDY/2":          {name: "spdy", version: "2"},
		"QUIC":            {name: "quic"},
		"unknown/proto/2": {name: "unknown", version: "proto/2"},
		"other":           {name: "other"},
	}

	for proto, want := range tests {
		name, version := netProtocol(proto)
		assert.Equal(t, want.name, name)
		assert.Equal(t, want.version, version)
	}
}
