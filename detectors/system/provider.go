// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package system // import "go.opentelemetry.io/contrib/detectors/system"

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	gofqdn "github.com/Showmax/go-fqdn"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Provider is the interface for querying system metadata.
// It is defined as an interface so it can be mocked in tests.
type Provider interface {
	// Hostname returns the OS hostname.
	Hostname() (string, error)
	// FQDN returns the fully qualified domain name.
	FQDN() (string, error)
	// LookupCNAME returns the canonical name for the current host.
	LookupCNAME() (string, error)
	// ReverseLookupHost does a reverse DNS query on the current host IP address.
	ReverseLookupHost() (string, error)
	// HostID returns the host unique identifier.
	HostID(ctx context.Context) (string, error)
	// HostArch returns the host CPU architecture.
	HostArch() (string, error)
	// HostIPs returns the host IP addresses (excluding loopback).
	HostIPs() ([]net.IP, error)
	// HostMACs returns the host MAC addresses (excluding loopback).
	HostMACs() ([]net.HardwareAddr, error)
	// OSType returns the operating system type.
	OSType() (string, error)
	// OSVersion returns the operating system version.
	OSVersion() (string, error)
	// OSDescription returns a human-readable OS description.
	OSDescription(ctx context.Context) (string, error)
	// CPUInfo returns CPU information.
	CPUInfo(ctx context.Context) ([]cpu.InfoStat, error)
}

// nameInfoProvider abstracts domain name resolution for testability.
type nameInfoProvider struct {
	osHostname  func() (string, error)
	lookupCNAME func(string) (string, error)
	lookupHost  func(string) ([]string, error)
	lookupAddr  func(string) ([]string, error)
}

func newNameInfoProvider() nameInfoProvider {
	return nameInfoProvider{
		osHostname:  os.Hostname,
		lookupCNAME: net.LookupCNAME,
		lookupHost:  net.LookupHost,
		lookupAddr:  net.LookupAddr,
	}
}

type systemMetadataProvider struct {
	nameInfoProvider
	newResource func(context.Context, ...resource.Option) (*resource.Resource, error)
}

// NewProvider returns a Provider that reads metadata from the local system.
func NewProvider() Provider {
	return &systemMetadataProvider{
		nameInfoProvider: newNameInfoProvider(),
		newResource:      resource.New,
	}
}

func (p *systemMetadataProvider) Hostname() (string, error) {
	return p.osHostname()
}

func (*systemMetadataProvider) FQDN() (string, error) {
	return gofqdn.FqdnHostname()
}

func (p *systemMetadataProvider) LookupCNAME() (string, error) {
	hostname, err := p.Hostname()
	if err != nil {
		return "", fmt.Errorf("LookupCNAME failed to get hostname: %w", err)
	}
	cname, err := p.lookupCNAME(hostname)
	if err != nil {
		return "", fmt.Errorf("LookupCNAME failed to get CNAME: %w", err)
	}
	return strings.TrimRight(cname, "."), nil
}

func (p *systemMetadataProvider) ReverseLookupHost() (string, error) {
	hostname, err := p.Hostname()
	if err != nil {
		return "", fmt.Errorf("ReverseLookupHost failed to get hostname: %w", err)
	}
	return p.hostnameToDomainName(hostname)
}

func (p *systemMetadataProvider) hostnameToDomainName(hostname string) (string, error) {
	ipAddresses, err := p.lookupHost(hostname)
	if err != nil {
		return "", fmt.Errorf("hostnameToDomainName failed to convert hostname to IP addresses: %w", err)
	}
	return p.reverseLookup(ipAddresses)
}

func (p *systemMetadataProvider) reverseLookup(ipAddresses []string) (string, error) {
	var lastErr error
	for _, ip := range ipAddresses {
		names, err := p.lookupAddr(ip)
		if err != nil {
			lastErr = err
			continue
		}
		return strings.TrimRight(names[0], "."), nil
	}
	return "", fmt.Errorf("reverseLookup failed to convert IP addresses to name: %w", lastErr)
}

func (p *systemMetadataProvider) fromOption(ctx context.Context, opt resource.Option, key string) (string, error) {
	res, err := p.newResource(ctx, opt)
	if err != nil {
		return "", fmt.Errorf("failed to obtain %q: %w", key, err)
	}
	iter := res.Iter()
	for iter.Next() {
		if iter.Attribute().Key == attribute.Key(key) {
			v := iter.Attribute().Value.Emit()
			if v == "" {
				return "", fmt.Errorf("empty %q", key)
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("failed to obtain %q", key)
}

func (p *systemMetadataProvider) HostID(ctx context.Context) (string, error) {
	return p.fromOption(ctx, resource.WithHostID(), string(semconv.HostIDKey))
}

func (p *systemMetadataProvider) OSDescription(ctx context.Context) (string, error) {
	return p.fromOption(ctx, resource.WithOSDescription(), string(semconv.OSDescriptionKey))
}

func (*systemMetadataProvider) HostArch() (string, error) {
	return goarchToHostArch(runtime.GOARCH), nil
}

func (*systemMetadataProvider) OSType() (string, error) {
	return goosToOSType(runtime.GOOS), nil
}

func (*systemMetadataProvider) OSVersion() (string, error) {
	info, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("OSVersion failed to get OS version: %w", err)
	}
	return info.PlatformVersion, nil
}

func (*systemMetadataProvider) HostIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, errAddr := iface.Addrs()
		if errAddr != nil {
			return nil, fmt.Errorf("failed to get addresses for interface %v: %w", iface.Name, errAddr)
		}
		for _, addr := range addrs {
			ip, _, parseErr := net.ParseCIDR(addr.String())
			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse address %q from interface %v: %w", addr, iface.Name, parseErr)
			}
			if ip.IsLoopback() {
				continue
			}
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

func (*systemMetadataProvider) HostMACs() ([]net.HardwareAddr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []net.HardwareAddr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		macs = append(macs, iface.HardwareAddr)
	}
	return macs, nil
}

func (*systemMetadataProvider) CPUInfo(ctx context.Context) ([]cpu.InfoStat, error) {
	return cpu.InfoWithContext(ctx)
}

// goosToOSType maps a runtime.GOOS value to the os.type semantic convention value.
func goosToOSType(goos string) string {
	switch goos {
	case "dragonfly":
		return "dragonflybsd"
	case "zos":
		return "z_os"
	}
	return goos
}

// goarchToHostArch maps a runtime.GOARCH value to the host.arch semantic convention value.
func goarchToHostArch(goarch string) string {
	switch goarch {
	case "arm":
		return "arm32"
	case "ppc64le":
		return "ppc64"
	case "386":
		return "x86"
	}
	return goarch
}
