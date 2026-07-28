// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const defaultEndpoint = "http://169.254.169.254/v1.json"

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataResponse is the JSON response from the Vultr instance metadata service.
type metadataResponse struct {
	Hostname     string         `json:"hostname"`
	InstanceID   string         `json:"instanceid"`
	InstanceV2ID string         `json:"instance-v2-id"`
	Region       metadataRegion `json:"region"`
}

type metadataRegion struct {
	RegionCode string `json:"regioncode"`
}

type config struct {
	filter attribute.Filter
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

// ResourceDetector collects resource information of Vultr Cloud Compute instances.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Vultr Cloud Compute instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
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
		endpoint: defaultEndpoint,
		cfg:      cfg,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// fetchMetadata queries the Vultr instance metadata endpoint. The returned
// boolean reports whether the process appears to be running on a Vultr
// instance: it is false when the metadata service cannot be reached or when
// something other than the Vultr metadata service answered the request.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*metadataResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on Vultr.
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the link-local address was answered by
		// something that is not the Vultr metadata service. Any other status
		// is a failure of the metadata service itself.
		onVultr := resp.StatusCode < 400 || resp.StatusCode > 499
		return nil, onVultr, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	var meta metadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, true, err
	}
	return &meta, true, nil
}

// Detect detects resource attributes of the Vultr Cloud Compute instance the
// process is running on. It returns an empty resource and no error when not
// running on a Vultr instance, and an error when the metadata service is
// reachable but does not return usable metadata. If the process is running on
// a Vultr instance but some attributes cannot be retrieved, a partial resource
// is returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, onVultr, err := d.fetchMetadata(ctx)
	if err != nil {
		if !onVultr {
			return resource.Empty(), nil
		}

		return nil, err
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
		// The metadata service reports region codes in upper case (e.g. "EWR").
		// Normalize to lower case so cloud.region matches the value the
		// collector's Vultr detector reports for the same instance.
		attrs = append(attrs, semconv.CloudRegion(strings.ToLower(meta.Region.RegionCode)))
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
