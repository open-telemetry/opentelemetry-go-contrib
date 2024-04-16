// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package host // import "go.opentelemetry.io/contrib/detectors/host"

import (
	"context"
	"net"
	"os"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type config struct {
	optInIPAddresses  bool
	optInMACAddresses bool
}

func newConfig(options ...Option) *config {
	c := &config{}
	for _, option := range options {
		option.apply(c)
	}

	return c
}

// Option applies an Azure VM detector configuration option.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (fn optionFunc) apply(c *config) {
	fn(c)
}

// WithIPAddresses adds the optional attribute `host.ip`.
func WithIPAddresses() Option {
	return optionFunc(func(c *config) {
		c.optInIPAddresses = true
	})
}

// WithMACAddresses adds the optional attribute `host.ip`.
func WithMACAddresses() Option {
	return optionFunc(func(c *config) {
		c.optInMACAddresses = true
	})
}

type resourceDetector struct {
	config *config
}

// NewResourceDetector returns a resource detector that will detect host resources.
func NewResourceDetector(opts ...Option) resource.Detector {
	c := newConfig(opts...)
	return &resourceDetector{config: c}
}

// Detect detects associated resources when running on a physical host.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
	}

	hostName, err := os.Hostname()
	if err == nil {
		attributes = append(attributes, semconv.HostName(hostName))
	}

	machineId, err := getHostId()
	if err == nil {
		attributes = append(attributes, semconv.HostID(machineId))
	}

	if detector.config.optInIPAddresses {
		ipAddresses := getIPAddresses()
		if len(ipAddresses) > 0 {
			attributes = append(attributes, semconv.HostIP(ipAddresses...))
		}
	}

	if detector.config.optInMACAddresses {
		macAddresses := getMACAddresses()
		if len(macAddresses) > 0 {
			attributes = append(attributes, semconv.HostMac(macAddresses...))
		}
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func getIPAddresses() []string {
	var ipAddresses []string

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipAddresses = append(ipAddresses, addr.String())
			}
		}
	}

	return ipAddresses
}

func getMACAddresses() []string {
	var macAddresses []string

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}

			macAddresses = append(macAddresses, iface.HardwareAddr.String())
		}
	}

	return macAddresses
}
