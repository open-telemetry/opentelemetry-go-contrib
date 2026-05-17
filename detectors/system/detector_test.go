// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// mockProvider implements Provider with configurable responses for testing.
type mockProvider struct {
	hostname         string
	hostnameErr      error
	fqdn             string
	fqdnErr          error
	cname            string
	cnameErr         error
	reverseLookup    string
	reverseLookupErr error
	hostID           string
	hostIDErr        error
	hostArch         string
	hostArchErr      error
	hostIPs          []net.IP
	hostIPsErr       error
	hostMACs         []net.HardwareAddr
	hostMACsErr      error
	osType           string
	osTypeErr        error
	osVersion        string
	osVersionErr     error
	osDescription    string
	osDescriptionErr error
	cpuInfo          []cpu.InfoStat
	cpuInfoErr       error
}

func (m *mockProvider) Hostname() (string, error)    { return m.hostname, m.hostnameErr }
func (m *mockProvider) FQDN() (string, error)        { return m.fqdn, m.fqdnErr }
func (m *mockProvider) LookupCNAME() (string, error) { return m.cname, m.cnameErr }

func (m *mockProvider) ReverseLookupHost() (string, error) {
	return m.reverseLookup, m.reverseLookupErr
}

func (m *mockProvider) HostID(_ context.Context) (string, error) { return m.hostID, m.hostIDErr }

func (m *mockProvider) HostArch() (string, error) { return m.hostArch, m.hostArchErr }

func (m *mockProvider) HostIPs() ([]net.IP, error) { return m.hostIPs, m.hostIPsErr }

func (m *mockProvider) HostMACs() ([]net.HardwareAddr, error) { return m.hostMACs, m.hostMACsErr }

func (m *mockProvider) OSType() (string, error) { return m.osType, m.osTypeErr }

func (m *mockProvider) OSVersion() (string, error) { return m.osVersion, m.osVersionErr }

func (m *mockProvider) OSDescription(_ context.Context) (string, error) {
	return m.osDescription, m.osDescriptionErr
}

func (m *mockProvider) CPUInfo(_ context.Context) ([]cpu.InfoStat, error) {
	return m.cpuInfo, m.cpuInfoErr
}

func newFullMockProvider() *mockProvider {
	mac1, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	return &mockProvider{
		hostname:      "myhost",
		fqdn:          "myhost.example.com",
		cname:         "myhost.example.com",
		reverseLookup: "myhost.example.com",
		hostID:        "abc-123",
		hostArch:      "amd64",
		hostIPs:       []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("10.0.0.1")},
		hostMACs:      []net.HardwareAddr{mac1},
		osType:        "linux",
		osVersion:     "22.04",
		osDescription: "Ubuntu 22.04.3 LTS",
		cpuInfo: []cpu.InfoStat{
			{
				VendorID:  "GenuineIntel",
				Family:    "6",
				Model:     "142",
				ModelName: "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz",
				Stepping:  10,
				CacheSize: 8192,
			},
		},
	}
}

func TestDetect_AllAttributes(t *testing.T) {
	p := newFullMockProvider()
	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"dns", "os"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	attrMap := attrsToMap(res)

	assert.Equal(t, "myhost.example.com", attrMap[string(semconv.HostNameKey)])
	assert.Equal(t, "abc-123", attrMap[string(semconv.HostIDKey)])
	assert.Equal(t, "amd64", attrMap[string(semconv.HostArchKey)])
	assert.Equal(t, "linux", attrMap[string(semconv.OSTypeKey)])
	assert.Equal(t, "22.04", attrMap[string(semconv.OSVersionKey)])
	assert.Equal(t, "Ubuntu 22.04.3 LTS", attrMap[string(semconv.OSDescriptionKey)])
	assert.Equal(t, "GenuineIntel", attrMap[string(semconv.HostCPUVendorIDKey)])
	assert.Equal(t, "6", attrMap[string(semconv.HostCPUFamilyKey)])
	assert.Equal(t, "142", attrMap[string(semconv.HostCPUModelIDKey)])
	assert.Equal(t, "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz", attrMap[string(semconv.HostCPUModelNameKey)])
	assert.Equal(t, "10", attrMap[string(semconv.HostCPUSteppingKey)])
}

func TestDetect_HostnameSourceFallback(t *testing.T) {
	p := newFullMockProvider()
	p.fqdnErr = errors.New("dns unavailable")

	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"dns", "os"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	attrMap := attrsToMap(res)
	assert.Equal(t, "myhost", attrMap[string(semconv.HostNameKey)])
}

func TestDetect_AllHostnameSourcesFail(t *testing.T) {
	p := newFullMockProvider()
	p.fqdnErr = errors.New("dns unavailable")
	p.hostnameErr = errors.New("os hostname unavailable")

	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"dns", "os"}},
	}

	_, err := d.Detect(t.Context())
	assert.ErrorContains(t, err, "all hostname sources failed")
}

func TestDetect_PartialResource(t *testing.T) {
	p := newFullMockProvider()
	p.hostIDErr = errors.New("no host ID")
	p.cpuInfoErr = errors.New("no CPU info")

	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"os"}},
	}

	res, err := d.Detect(t.Context())
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	attrMap := attrsToMap(res)
	assert.Equal(t, "myhost", attrMap[string(semconv.HostNameKey)])
	assert.NotContains(t, attrMap, string(semconv.HostIDKey))
	assert.NotContains(t, attrMap, string(semconv.HostCPUVendorIDKey))
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	p := newFullMockProvider()
	d := &ResourceDetector{
		provider: p,
		cfg: config{
			hostnameSources: []string{"os"},
			filter: func(kv attribute.KeyValue) bool {
				return kv.Key == semconv.HostNameKey || kv.Key == semconv.OSTypeKey
			},
		},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	attrMap := attrsToMap(res)
	assert.Equal(t, "myhost", attrMap[string(semconv.HostNameKey)])
	assert.Equal(t, "linux", attrMap[string(semconv.OSTypeKey)])
	assert.NotContains(t, attrMap, string(semconv.HostIDKey))
	assert.NotContains(t, attrMap, string(semconv.HostArchKey))
}

func TestDetect_HostnameSources_CName(t *testing.T) {
	p := newFullMockProvider()
	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"cname"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "myhost.example.com", attrsToMap(res)[string(semconv.HostNameKey)])
}

func TestDetect_HostnameSources_ReverseLookup(t *testing.T) {
	p := newFullMockProvider()
	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"lookup"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "myhost.example.com", attrsToMap(res)[string(semconv.HostNameKey)])
}

func TestDetect_MACAddressIEEEFormat(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	p := newFullMockProvider()
	p.hostMACs = []net.HardwareAddr{mac}

	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"os"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	iter := res.Iter()
	for iter.Next() {
		if iter.Attribute().Key == semconv.HostMacKey {
			vals := iter.Attribute().Value.AsStringSlice()
			require.Len(t, vals, 1)
			assert.Equal(t, "AA-BB-CC-DD-EE-FF", vals[0])
		}
	}
}

func TestDetect_CPUModelIDSkippedWhenEmpty(t *testing.T) {
	p := newFullMockProvider()
	p.cpuInfo[0].Model = ""

	d := &ResourceDetector{
		provider: p,
		cfg:      config{hostnameSources: []string{"os"}},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	attrMap := attrsToMap(res)
	assert.NotContains(t, attrMap, string(semconv.HostCPUModelIDKey))
}

func TestNewResourceDetector_PanicsOnInvalidSource(t *testing.T) {
	assert.Panics(t, func() {
		NewResourceDetector(WithHostnameSources("invalid"))
	})
}

func TestToIEEERA(t *testing.T) {
	mac, err := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	require.NoError(t, err)
	assert.Equal(t, "AA-BB-CC-DD-EE-FF", toIEEERA(mac))
}

func TestGoosToOSType(t *testing.T) {
	cases := []struct{ in, want string }{
		{"linux", "linux"},
		{"darwin", "darwin"},
		{"windows", "windows"},
		{"dragonfly", "dragonflybsd"},
		{"zos", "z_os"},
		{"freebsd", "freebsd"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, goosToOSType(tc.in), "input: %s", tc.in)
	}
}

func TestGoarchToHostArch(t *testing.T) {
	cases := []struct{ in, want string }{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"arm", "arm32"},
		{"ppc64le", "ppc64"},
		{"386", "x86"},
		{"s390x", "s390x"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, goarchToHostArch(tc.in), "input: %s", tc.in)
	}
}

// attrsToMap converts a resource's attributes to a flat string map for easy assertion.
// For slice-valued attributes only the raw string representation is stored.
func attrsToMap(res *resource.Resource) map[string]string {
	m := make(map[string]string)
	iter := res.Iter()
	for iter.Next() {
		kv := iter.Attribute()
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}
