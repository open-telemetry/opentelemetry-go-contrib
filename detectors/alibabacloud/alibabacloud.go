// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package alibabacloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
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

	// Alibaba Cloud network identifiers have no semantic-conventions
	// counterparts in the current spec.
	vpcIDKey     = attribute.Key("alibaba.cloud.vpc.id")
	vswitchIDKey = attribute.Key("alibaba.cloud.vswitch.id")
)

// errMissing reports that the metadata service answered without the requested
// item: the process is either not running on Alibaba Cloud or the item does
// not apply to the current instance.
var errMissing = errors.New("metadata item missing")

// errUnreachable reports that the metadata service could not be reached.
var errUnreachable = errors.New("metadata service unreachable")

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// ResourceDetector collects resource information of Alibaba Cloud ECS
// instances and of containers running on ECI, including pods on Alibaba Cloud
// Container Compute Service (ACS).
type ResourceDetector struct {
	endpoint string
	client   *http.Client
}

// NewResourceDetector returns a [ResourceDetector] that detects resource
// attributes on Alibaba Cloud ECS instances and ECI containers.
func NewResourceDetector() *ResourceDetector {
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (100.100.100.200) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic.
	return &ResourceDetector{
		endpoint: defaultEndpoint,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: &http.Transport{Proxy: nil},
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
	// Prefer the hardened IMDSv2 token flow and fall back to IMDSv1 when the
	// token endpoint is unavailable, e.g. on ECI containers.
	token, err := d.fetchToken(ctx)
	if err != nil {
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
		semconv.CloudResourceID(instanceID),
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
	fetchAdd("mac", func(v string) attribute.KeyValue { return semconv.HostMac(v) })
	fetchAdd("vpc-id", vpcIDKey.String)
	fetchAdd("vswitch-id", vswitchIDKey.String)

	var ips []string
	fetchIP := func(path string) {
		v, err := d.fetchMeta(ctx, token, path)
		switch {
		case errors.Is(err, errMissing):
		case err != nil:
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		case v != "" && !slices.Contains(ips, v):
			ips = append(ips, v)
		}
	}
	fetchIP("private-ipv4")
	fetchIP("public-ipv4")
	fetchIP("eipv4")
	if len(ips) > 0 {
		attrs = append(attrs, semconv.HostIP(ips...))
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %v", resource.ErrPartialResource, errs)
	}
	return res, nil
}

// fetchToken retrieves an IMDSv2 session token. A non-nil error means the
// token flow is unavailable and the caller should fall back to IMDSv1.
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
