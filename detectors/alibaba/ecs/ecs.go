// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ecs

import (
	"context"
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

//nolint:gosec // G101 false positives: these are URL paths and HTTP header names, not credentials.
const (
	defaultEndpoint = "http://100.100.100.200"

	tokenPath        = "/latest/api/token"
	metadataBasePath = "/latest/meta-data/"

	// tokenTTLHeader carries the requested lifetime of a metadata token.
	tokenTTLHeader = "X-aliyun-ecs-metadata-token-ttl-seconds"
	// tokenHeader carries the metadata token on metadata requests.
	tokenHeader = "X-aliyun-ecs-metadata-token"
	// tokenTTL is the requested token lifetime in seconds. The metadata service
	// accepts 1 to 21600. A token is used for the duration of a single Detect
	// call and never cached, so it is deliberately kept short lived.
	tokenTTL = "60"

	// maxResponseSize bounds how much of a metadata response is read. Every
	// value the metadata service returns is a short scalar, so anything larger
	// is a sign the request was answered by something else.
	maxResponseSize = 1 << 20
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataFields are the metadata service paths this detector reads and the
// attribute each one provides. Alibaba Cloud ECS serves a single plain text
// scalar per path; there is no aggregate document to fetch instead.
var metadataFields = []struct {
	path string
	attr func(string) attribute.KeyValue
}{
	{"owner-account-id", semconv.CloudAccountID},
	{"region-id", semconv.CloudRegion},
	{"zone-id", semconv.CloudAvailabilityZone},
	{"instance-id", semconv.HostID},
	{"hostname", semconv.HostName},
	{"image-id", semconv.HostImageID},
	{"instance/instance-type", semconv.HostType},
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

// ResourceDetector collects resource information of Alibaba Cloud ECS instances.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Alibaba Cloud ECS instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// an Alibaba Cloud internal address (100.100.100.200) that must never be
	// reached via an HTTP(S) proxy: doing so could leak instance metadata or
	// break detection in environments where users set HTTP_PROXY/HTTPS_PROXY
	// for outbound traffic.
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

// fetchToken requests a token for the instance metadata service. The returned
// boolean reports whether the process appears to be running on an Alibaba Cloud
// ECS instance: it is false when the metadata service cannot be reached or when
// something other than the metadata service answered the request.
func (d *ResourceDetector) fetchToken(ctx context.Context) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, d.endpoint+tokenPath, http.NoBody)
	if err != nil {
		return "", false, err
	}
	req.Header.Set(tokenTTLHeader, tokenTTL)

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on Alibaba Cloud ECS.
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the metadata address was answered by something
		// that is not the Alibaba Cloud ECS metadata service. Any other status
		// is a failure of the metadata service itself.
		onECS := resp.StatusCode < 400 || resp.StatusCode > 499
		return "", onECS, fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", true, err
	}

	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", true, errors.New("token request returned an empty token")
	}
	return token, true, nil
}

// fetchMetadata reads the value the metadata service serves at path.
func (d *ResourceDetector) fetchMetadata(ctx context.Context, token, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+metadataBasePath+path, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set(tokenHeader, token)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// Detect detects resource attributes of the Alibaba Cloud ECS instance the
// process is running on. It returns an empty resource and no error when not
// running on an ECS instance, and an error when the metadata service is
// reachable but does not return usable metadata. If the process is running on
// an ECS instance but some attributes cannot be retrieved, a partial resource
// is returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	token, onECS, err := d.fetchToken(ctx)
	if err != nil {
		if !onECS {
			return resource.Empty(), nil
		}

		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
	}

	var errs []error
	for _, f := range metadataFields {
		val, err := d.fetchMetadata(ctx, token, f.path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.path, err))
			continue
		}
		if val == "" {
			errs = append(errs, fmt.Errorf("%s: not present in metadata", f.path))
			continue
		}
		attrs = append(attrs, f.attr(val))
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
