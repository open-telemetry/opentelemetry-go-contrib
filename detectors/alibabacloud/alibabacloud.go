// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package alibabacloud

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

const (
	defaultEndpoint = "http://100.100.100.200"

	tokenEndpoint = "/latest/api/token" //nolint:gosec // False positive G101: HTTP endpoint path, not a credential.
	metaPrefix    = "/latest/meta-data/"

	tokenTTLHeader = "X-aliyun-ecs-metadata-token-ttl-seconds" //nolint:gosec // False positive G101: HTTP header name, not a credential.
	tokenHeader    = "X-aliyun-ecs-metadata-token"             //nolint:gosec // False positive G101: HTTP header name, not a credential.
	tokenTTL       = "21600"

	// Metadata values are short identifiers (IDs, IPs, hostnames). Bound the
	// response body size so a misbehaving service cannot amplify memory use.
	maxBodySize = 4 << 10
)

// errMissing reports that the metadata service answered without the requested
// item: the process is either not running on Alibaba Cloud or the item does
// not apply to the current instance.
var errMissing = errors.New("metadata item missing")

// errUnreachable reports that the metadata service could not be reached.
var errUnreachable = errors.New("metadata service unreachable")

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

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

// ResourceDetector collects resource information of Alibaba Cloud ECS
// instances and of containers running on ECI, including pods on Alibaba Cloud
// Container Compute Service (ACS).
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [ResourceDetector] that detects resource
// attributes on Alibaba Cloud ECS instances and ECI containers.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (100.100.100.200) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic. The idle timeout bounds how long a discarded detector keeps its
	// connections to the metadata service open.
	return &ResourceDetector{
		endpoint: defaultEndpoint,
		cfg:      cfg,
		client: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				Proxy:           nil,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

// Detect detects resource attributes of the Alibaba Cloud instance the
// process is running on. It returns an empty resource and no error when not
// running on Alibaba Cloud, and an error when the metadata service is
// reachable but fails. If the process is running on Alibaba Cloud but some
// attributes cannot be retrieved, a partial resource is returned together
// with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	// Prefer the hardened IMDSv2 token flow. Only fall back to IMDSv1 when the
	// token endpoint answers with an HTTP error status, e.g. on ECI containers.
	// An unreachable token endpoint means the metadata service itself is
	// unreachable, so detection stops instead of issuing a second doomed
	// request that would double the timeout.
	token, err := d.fetchToken(ctx)
	if err != nil {
		if errors.Is(err, errUnreachable) {
			return resource.Empty(), nil
		}
		token = ""
	}

	// The instance ID anchors detection: if it cannot be read, the process is
	// not running on Alibaba Cloud as far as this detector is concerned.
	instanceID, err := d.fetchMeta(ctx, token, "instance-id")
	if err != nil {
		if errors.Is(err, errMissing) || errors.Is(err, errUnreachable) {
			return resource.Empty(), nil
		}
		return nil, err
	}
	if instanceID == "" {
		return nil, errors.New("metadata service returned an empty instance-id")
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
		semconv.HostID(instanceID),
	}

	var errs []error

	fetchAdd := func(path string, set func(string) attribute.KeyValue) {
		v, err := d.fetchMeta(ctx, token, path)
		switch {
		case errors.Is(err, errMissing):
			// Not every metadata item applies to every instance.
		case err != nil:
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		case v != "":
			attrs = append(attrs, set(v))
		}
	}

	fetchAdd("region-id", semconv.CloudRegion)
	fetchAdd("zone-id", semconv.CloudAvailabilityZone)
	fetchAdd("owner-account-id", semconv.CloudAccountID)
	fetchAdd("hostname", semconv.HostName)
	fetchAdd("instance/instance-type", semconv.HostType)
	fetchAdd("image-id", semconv.HostImageID)

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

// fetchToken retrieves an IMDSv2 session token. A non-nil error means the
// token flow is unavailable: the caller falls back to IMDSv1 unless the error
// reports [errUnreachable], in which case there is no metadata service to
// talk to at all.
func (d *ResourceDetector) fetchToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, d.endpoint+tokenEndpoint, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set(tokenTTLHeader, tokenTTL)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", errors.New("token response is empty")
	}
	return token, nil
}

// fetchMeta requests a single metadata item, e.g. "instance-id". A 403 or 404
// response is reported as [errMissing], a transport failure as [errUnreachable],
// and any other non-success status as an error naming the item and status.
func (d *ResourceDetector) fetchMeta(ctx context.Context, token, item string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+metaPrefix+item, http.NoBody)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set(tokenHeader, token)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errUnreachable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusNotFound:
		return "", errMissing
	default:
		return "", fmt.Errorf("metadata request for %q returned status %d", item, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
