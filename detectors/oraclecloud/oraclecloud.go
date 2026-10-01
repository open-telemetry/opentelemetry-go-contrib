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
	authHeader      = "Bearer Oracle"

	maxBodySize = 1 * 1024 * 1024
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

type computeMetadata struct {
	HostID             string           `json:"id"`
	HostDisplayName    string           `json:"displayName"`
	HostType           string           `json:"shape"`
	CanonicalRegionID  string           `json:"canonicalRegionName"` // Primary
	RegionID           string           `json:"region"`              // Fallback
	AvailabilityDomain string           `json:"availabilityDomain"`
	Metadata           instanceMetadata `json:"metadata"`
}

type instanceMetadata struct {
	OKEClusterDisplayName string `json:"oke-cluster-display-name"`
	Realm                 string `json:"realm"`
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

// ResourceDetector collects resource information of Oracle Cloud Infrastructure (OCI) instances.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
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
		cfg:      cfg,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// fetchMetadata queries the OCI instance metadata endpoint.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*computeMetadata, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		onOracleCloud := resp.StatusCode < 400 || resp.StatusCode > 499
		return nil, onOracleCloud, fmt.Errorf("received non-OK response from Oracle Cloud IMDS: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, true, fmt.Errorf("failed to read Oracle Cloud IMDS response: %w", err)
	}

	var meta computeMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, true, fmt.Errorf("failed to decode Oracle Cloud IMDS response: %w", err)
	}

	return &meta, true, nil
}

// Detect detects resource attributes of the Oracle Cloud Infrastructure instance
// the process is running on. It returns an empty resource and no error when not
// running on an Oracle Cloud instance. If running on Oracle Cloud but some
// attributes cannot be retrieved, a partial resource is returned together with
// [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, onOracleCloud, err := d.fetchMetadata(ctx)
	if err != nil {
		if !onOracleCloud {
			return resource.Empty(), nil
		}
		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderOracleCloud,
	}

	if meta.Metadata.OKEClusterDisplayName != "" {
		attrs = append(attrs, semconv.CloudPlatformOracleCloudOKE, semconv.K8SClusterName(meta.Metadata.OKEClusterDisplayName))
	} else {
		attrs = append(attrs, semconv.CloudPlatformOracleCloudCompute)
	}

	var errs []error

	if meta.HostID == "" {
		errs = append(errs, errors.New("host ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostID(meta.HostID))
	}

	if meta.HostDisplayName == "" {
		errs = append(errs, errors.New("hostname: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostName(meta.HostDisplayName))
	}

	if meta.HostType == "" {
		errs = append(errs, errors.New("host type: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostType(meta.HostType))
	}

	region := meta.CanonicalRegionID
	if region == "" {
		region = meta.RegionID
	}
	if region == "" {
		errs = append(errs, errors.New("region: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudRegion(region))
	}

	if meta.AvailabilityDomain == "" {
		errs = append(errs, errors.New("availability zone: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudAvailabilityZone(meta.AvailabilityDomain))
	}

	if meta.Metadata.Realm != "" {
		attrs = append(attrs, semconv.OracleCloudRealm(meta.Metadata.Realm))
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
