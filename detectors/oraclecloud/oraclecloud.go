// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

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

const (
	defaultEndpoint = "http://169.254.169.254/opc/v2/instance/"
	maxBodySize     = 1 << 20
)

var _ resource.Detector = (*ResourceDetector)(nil)

var errRedirectRejected = errors.New("redirects are not allowed for OCI metadata")

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

// ResourceDetector collects resource information from OCI instance metadata.
type ResourceDetector struct {
	endpoint string
	client   *http.Client
	cfg      config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Oracle Cloud Infrastructure instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	transport := &http.Transport{Proxy: nil}
	return &ResourceDetector{
		endpoint: defaultEndpoint,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errRedirectRejected
			},
		},
		cfg: cfg,
	}
}

type metadataResponse struct {
	ID                 string            `json:"id"`
	Hostname           string            `json:"hostname"`
	AvailabilityDomain string            `json:"availabilityDomain"`
	CanonicalRegion    string            `json:"canonicalRegionName"`
	Shape              string            `json:"shape"`
	RegionInfo         regionInfo        `json:"regionInfo"`
	Metadata           map[string]string `json:"metadata"`
}

type regionInfo struct {
	RealmKey string `json:"realmKey"`
}

// Detect detects resource attributes of the Oracle Cloud Infrastructure
// instance the process is running on. It returns an empty resource and no error
// when metadata is unreachable or a non-throttling 4xx response indicates the
// process is not on OCI. Valid but incomplete metadata returns a partial
// resource with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, err := d.fetchMetadata(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	if meta == nil {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderOracleCloud,
		platform(meta),
	}
	if meta.CanonicalRegion != "" {
		attrs = append(attrs, semconv.CloudRegion(meta.CanonicalRegion))
	}
	if meta.AvailabilityDomain != "" {
		attrs = append(attrs, semconv.CloudAvailabilityZone(meta.AvailabilityDomain))
	}
	if meta.Hostname != "" {
		attrs = append(attrs, semconv.HostName(meta.Hostname))
	}
	if meta.Shape != "" {
		attrs = append(attrs, semconv.HostType(meta.Shape))
	}
	if clusterName := meta.Metadata["oke-cluster-display-name"]; clusterName != "" {
		attrs = append(attrs, semconv.K8SClusterName(clusterName))
	}
	if realm := realm(meta); realm != "" {
		attrs = append(attrs, semconv.OracleCloudRealm(realm))
	}

	if meta.ID == "" {
		attrs = filterAttributes(attrs, d.cfg.filter)
		res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
		return res, fmt.Errorf("%w: instance ID is not present in metadata", resource.ErrPartialResource)
	}
	attrs = append(attrs, semconv.HostID(meta.ID), semconv.CloudResourceID(meta.ID))

	attrs = filterAttributes(attrs, d.cfg.filter)
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*metadataResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer Oracle")
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if errors.Is(err, errRedirectRejected) {
			return nil, err
		}
		return nil, nil
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	case resp.StatusCode >= 400 && resp.StatusCode <= 499:
		return nil, nil
	default:
		return nil, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	if len(body) > maxBodySize {
		return nil, fmt.Errorf("metadata response exceeds %d bytes", maxBodySize)
	}

	var meta metadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &meta, nil
}

func platform(meta *metadataResponse) attribute.KeyValue {
	if meta.Metadata["oke-cluster-display-name"] != "" {
		return semconv.CloudPlatformOracleCloudOKE
	}
	return semconv.CloudPlatformOracleCloudCompute
}

func realm(meta *metadataResponse) string {
	if meta.RegionInfo.RealmKey != "" {
		return meta.RegionInfo.RealmKey
	}
	return meta.Metadata["realm"]
}

func filterAttributes(attrs []attribute.KeyValue, filter attribute.Filter) []attribute.KeyValue {
	if filter == nil {
		return attrs
	}
	filtered := attrs[:0]
	for _, kv := range attrs {
		if filter(kv) {
			filtered = append(filtered, kv)
		}
	}
	return filtered
}
