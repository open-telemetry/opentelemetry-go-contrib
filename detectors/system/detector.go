// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package system // import "go.opentelemetry.io/contrib/detectors/system"

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Compile-time assertion that ResourceDetector implements resource.Detector.
var _ resource.Detector = (*ResourceDetector)(nil)

// hostnameSource is a function that resolves the hostname via a specific strategy.
type hostnameSource func(p Provider) (string, error)

var hostnameSourcesMap = map[string]hostnameSource{
	"os": func(p Provider) (string, error) {
		h, err := p.Hostname()
		if err != nil {
			return "", fmt.Errorf("failed getting OS hostname: %w", err)
		}
		return h, nil
	},
	"dns": func(p Provider) (string, error) {
		h, err := p.FQDN()
		if err != nil {
			return "", fmt.Errorf("failed getting FQDN: %w", err)
		}
		return h, nil
	},
	"cname": func(p Provider) (string, error) {
		h, err := p.LookupCNAME()
		if err != nil {
			return "", fmt.Errorf("failed getting CNAME: %w", err)
		}
		return h, nil
	},
	"lookup": func(p Provider) (string, error) {
		h, err := p.ReverseLookupHost()
		if err != nil {
			return "", fmt.Errorf("failed doing reverse DNS lookup: %w", err)
		}
		return h, nil
	},
}

// config holds configuration for [ResourceDetector].
type config struct {
	hostnameSources []string
	filter          attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithHostnameSources sets the ordered list of strategies used to resolve the
// host name. Each value must be one of: "dns", "os", "cname", "lookup".
// The first successful resolution is used. Defaults to ["dns", "os"].
func WithHostnameSources(sources ...string) Option {
	return optionFunc(func(c *config) {
		c.hostnameSources = sources
	})
}

// WithAttributeFilter sets a filter controlling which detected attributes are
// included in the returned resource. Only attributes for which the filter
// returns true are included. By default all detected attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) {
		c.filter = filter
	})
}

// ResourceDetector detects host- and OS-level resource attributes from the
// local system.
type ResourceDetector struct {
	provider Provider
	cfg      config
}

// NewResourceDetector returns a [resource.Detector] that detects system
// resource attributes from the local host.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{
		hostnameSources: []string{"dns", "os"},
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	for _, src := range cfg.hostnameSources {
		if _, ok := hostnameSourcesMap[src]; !ok {
			panic(fmt.Sprintf("system detector: unknown hostname source %q (valid: dns, os, cname, lookup)", src))
		}
	}
	return &ResourceDetector{
		provider: NewProvider(),
		cfg:      cfg,
	}
}

// toIEEERA converts a MAC address to IEEE RA format (uppercase, hyphen-separated).
func toIEEERA(mac net.HardwareAddr) string {
	return strings.ToUpper(strings.ReplaceAll(mac.String(), ":", "-"))
}

// Detect detects resource attributes of the local system.
//
// If the hostname cannot be resolved from any of the configured sources,
// a non-nil error is returned. If some optional attributes cannot be detected,
// a partial resource together with [resource.ErrPartialResource] is returned.
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	// hostname (required — try configured sources in order)
	hostname, err := d.resolveHostname()
	if err != nil {
		return resource.Empty(), err
	}
	attrs = append(attrs, semconv.HostName(hostname))

	// host.id
	if hostID, err := d.provider.HostID(ctx); err == nil {
		attrs = append(attrs, semconv.HostID(hostID))
	} else {
		errs = append(errs, fmt.Errorf("host.id: %w", err))
	}

	// host.arch
	if hostArch, err := d.provider.HostArch(); err == nil {
		attrs = append(attrs, semconv.HostArchKey.String(hostArch))
	} else {
		errs = append(errs, fmt.Errorf("host.arch: %w", err))
	}

	// host.ip (single multi-value attribute)
	if hostIPs, err := d.provider.HostIPs(); err == nil {
		ipStrs := make([]string, len(hostIPs))
		for i, ip := range hostIPs {
			ipStrs[i] = ip.String()
		}
		attrs = append(attrs, semconv.HostIP(ipStrs...))
	} else {
		errs = append(errs, fmt.Errorf("host.ip: %w", err))
	}

	// host.mac (single multi-value attribute)
	if hostMACs, err := d.provider.HostMACs(); err == nil {
		macStrs := make([]string, len(hostMACs))
		for i, mac := range hostMACs {
			macStrs[i] = toIEEERA(mac)
		}
		attrs = append(attrs, semconv.HostMac(macStrs...))
	} else {
		errs = append(errs, fmt.Errorf("host.mac: %w", err))
	}

	// os.type
	if osType, err := d.provider.OSType(); err == nil {
		attrs = append(attrs, semconv.OSTypeKey.String(osType))
	} else {
		errs = append(errs, fmt.Errorf("os.type: %w", err))
	}

	// os.version
	if osVersion, err := d.provider.OSVersion(); err == nil {
		attrs = append(attrs, semconv.OSVersion(osVersion))
	} else {
		errs = append(errs, fmt.Errorf("os.version: %w", err))
	}

	// os.description
	if osDesc, err := d.provider.OSDescription(ctx); err == nil {
		attrs = append(attrs, semconv.OSDescription(osDesc))
	} else {
		errs = append(errs, fmt.Errorf("os.description: %w", err))
	}

	// host.cpu.*
	if cpuInfos, err := d.provider.CPUInfo(ctx); err == nil && len(cpuInfos) > 0 {
		c := cpuInfos[0]
		attrs = append(attrs, semconv.HostCPUVendorID(c.VendorID))
		attrs = append(attrs, semconv.HostCPUFamily(c.Family))
		if c.Model != "" {
			attrs = append(attrs, semconv.HostCPUModelID(c.Model))
		}
		attrs = append(attrs, semconv.HostCPUModelName(c.ModelName))
		attrs = append(attrs, semconv.HostCPUStepping(strconv.Itoa(int(c.Stepping))))
		attrs = append(attrs, semconv.HostCPUCacheL2Size(int(c.CacheSize)))
	} else if err != nil {
		errs = append(errs, fmt.Errorf("host.cpu.*: %w", err))
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %w", resource.ErrPartialResource, errors.Join(errs...))
	}
	return res, nil
}

func (d *ResourceDetector) resolveHostname() (string, error) {
	var lastErr error
	for _, src := range d.cfg.hostnameSources {
		h, err := hostnameSourcesMap[src](d.provider)
		if err == nil {
			return h, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all hostname sources failed: %w", lastErr)
}
