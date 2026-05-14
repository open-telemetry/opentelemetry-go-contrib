// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vultr // import "go.opentelemetry.io/contrib/detectors/vultr"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const defaultEndpoint = "http://169.254.169.254/v1.json"

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataResponse is the JSON response from the Vultr instance metadata service.
type metadataResponse struct {
	Hostname     string `json:"hostname"`
	InstanceID   string `json:"instanceid"`
	InstanceV2ID string `json:"instance-v2-id"`
	Region       struct {
		RegionCode string `json:"regioncode"`
	} `json:"region"`
}

type config struct {
	filter   attribute.Filter
	endpoint string
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// WithEndpoint overrides the metadata service endpoint. Intended for testing.
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(c *config) { c.endpoint = endpoint })
}

// ResourceDetector collects resource information of Vultr Cloud Compute instances.
type ResourceDetector struct {
	cfg    config
	client *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Vultr Cloud Compute instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{endpoint: defaultEndpoint}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (169.254.169.254) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic.
	transport := &http.Transport{Proxy: nil}
	return &ResourceDetector{
		cfg: cfg,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// fetchMetadata queries the Vultr instance metadata endpoint.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*metadataResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.cfg.endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meta metadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Detect detects resource attributes of the Vultr Cloud Compute instance the
// process is running on. It returns an empty resource and no error when not
// running on a Vultr instance. If the process is running on a Vultr instance
// but some attributes cannot be retrieved, a partial resource is returned
// together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, err := d.fetchMetadata(ctx)
	if err != nil {
		// Not on Vultr or metadata service unreachable — return empty, no error.
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderVultr,
		semconv.CloudPlatformVultrCloudCompute,
	}

	var errs []error

	// Prefer the v2 UUID; fall back to legacy instanceid.
	instanceID := meta.InstanceV2ID
	if instanceID == "" {
		instanceID = meta.InstanceID
	}
	if instanceID == "" {
		errs = append(errs, errors.New("instance ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostID(instanceID))
	}

	if meta.Hostname == "" {
		errs = append(errs, errors.New("hostname: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostName(meta.Hostname))
	}

	if meta.Region.RegionCode == "" {
		errs = append(errs, errors.New("region: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudRegion(meta.Region.RegionCode))
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
		return res, fmt.Errorf("%w: %v", resource.ErrPartialResource, errs)
	}
	return res, nil
}
