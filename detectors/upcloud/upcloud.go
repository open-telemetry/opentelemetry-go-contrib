// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloud

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
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const defaultEndpoint = "http://169.254.169.254/metadata/v1.json"

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataResponse is the JSON response from the UpCloud Instance Metadata
// Service. Only the fields used to build resource attributes are declared; the
// document also carries network, storage, tags, and user data.
type metadataResponse struct {
	CloudName  string `json:"cloud_name"`
	Hostname   string `json:"hostname"`
	InstanceID string `json:"instance_id"`
	Region     string `json:"region"`
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

// ResourceDetector collects resource information of UpCloud Cloud Servers.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on UpCloud Cloud Servers.
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

// fetchMetadata queries the UpCloud instance metadata endpoint. The returned
// boolean reports whether the process appears to be running on an UpCloud
// Cloud Server: it is false when the metadata service cannot be reached or when
// something other than the UpCloud metadata service answered the request.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*metadataResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on UpCloud.
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the link-local address was answered by
		// something that is not the UpCloud metadata service. Any other status
		// is a failure of the metadata service itself.
		onUpCloud := resp.StatusCode < 400 || resp.StatusCode > 499
		return nil, onUpCloud, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
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

// Detect detects resource attributes of the UpCloud Cloud Server the process is
// running on. It returns an empty resource and no error when not running on an
// UpCloud Cloud Server, and an error when the metadata service is reachable but
// does not return usable metadata. If the process is running on an UpCloud
// Cloud Server but some attributes cannot be retrieved, a partial resource is
// returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, onUpCloud, err := d.fetchMetadata(ctx)
	if err != nil {
		if !onUpCloud {
			return resource.Empty(), nil
		}

		return nil, err
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	// semconv has no CloudProvider enum value for UpCloud, so the value
	// reported by the metadata service ("upcloud") is used as-is, matching the
	// collector's UpCloud detector.
	// TODO: Use the semantic convention constant when one is added.
	if meta.CloudName == "" {
		errs = append(errs, errors.New("cloud name: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudProviderKey.String(meta.CloudName))
	}

	if meta.Region == "" {
		errs = append(errs, errors.New("region: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudRegion(meta.Region))
	}

	if meta.InstanceID == "" {
		errs = append(errs, errors.New("instance ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostID(meta.InstanceID))
	}

	if meta.Hostname == "" {
		errs = append(errs, errors.New("hostname: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostName(meta.Hostname))
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
