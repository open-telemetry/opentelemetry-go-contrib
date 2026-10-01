// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cvm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	defaultEndpoint = "http://metadata.tencentyun.com/latest/meta-data"

	// maxResponseSize bounds how much of a metadata response is read. Every
	// document served by the metadata service is a short string, so a larger
	// response means something other than the metadata service answered.
	maxResponseSize = 1 << 20
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataKey is a single document served by the Tencent Cloud CVM instance
// metadata service and the resource attribute built from it.
type metadataKey struct {
	path string
	attr func(string) attribute.KeyValue
}

// metadataKeys are the documents Detect fetches. The metadata service has no
// combined document, so each is a separate request. The order is also the
// order failures are reported in.
var metadataKeys = []metadataKey{
	{"app-id", semconv.CloudAccountID},
	{"placement/region", semconv.CloudRegion},
	{"placement/zone", semconv.CloudAvailabilityZone},
	{"instance-id", semconv.HostID},
	{"instance-name", semconv.HostName},
	{"instance/image-id", semconv.HostImageID},
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

// ResourceDetector collects resource information of Tencent Cloud CVM instances.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Tencent Cloud CVM instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// only meant to be reached from within the instance and must never be
	// requested through an HTTP(S) proxy: doing so could leak instance
	// metadata or break detection in environments where users set
	// HTTP_PROXY/HTTPS_PROXY for outbound traffic.
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

// fetch retrieves a single metadata document. The returned boolean reports
// whether the process appears to be running on a Tencent Cloud CVM instance:
// it is false when the metadata service cannot be reached or when something
// other than the metadata service answered the request.
func (d *ResourceDetector) fetch(ctx context.Context, path string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/"+path, http.NoBody)
	if err != nil {
		return "", false, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on Tencent Cloud.
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the metadata host was answered by something
		// that is not the Tencent Cloud metadata service. Any other status is
		// a failure of the metadata service itself.
		onTencent := resp.StatusCode < 400 || resp.StatusCode > 499
		return "", onTencent, fmt.Errorf("metadata request for %q returned status %d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return "", true, err
	}
	if len(body) > maxResponseSize {
		return "", true, fmt.Errorf("metadata response for %q exceeds %d bytes", path, maxResponseSize)
	}

	// Responses are plain text. Trim so that trailing whitespace served by the
	// metadata service does not end up in the attribute value.
	return strings.TrimSpace(string(body)), true, nil
}

// metadata fetches every document in metadataKeys concurrently and returns
// their values in the same order.
//
// Detection is all-or-nothing, matching the collector's Tencent Cloud CVM
// detector: if any document cannot be retrieved, the first failure in
// metadataKeys order is returned along with whether the process still appears
// to be running on a CVM instance.
func (d *ResourceDetector) metadata(ctx context.Context) ([]string, bool, error) {
	values := make([]string, len(metadataKeys))
	onTencent := make([]bool, len(metadataKeys))
	errs := make([]error, len(metadataKeys))

	var wg sync.WaitGroup
	wg.Add(len(metadataKeys))
	for i, key := range metadataKeys {
		go func() {
			defer wg.Done()
			values[i], onTencent[i], errs[i] = d.fetch(ctx, key.path)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			return nil, onTencent[i], err
		}
	}
	return values, true, nil
}

// Detect detects resource attributes of the Tencent Cloud CVM instance the
// process is running on. It returns an empty resource and no error when not
// running on a CVM instance, and an error when the metadata service is
// reachable but does not return usable metadata. If the metadata service
// answers every request but omits some values, a partial resource is returned
// together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	values, onTencent, err := d.metadata(ctx)
	if err != nil {
		if !onTencent {
			return resource.Empty(), nil
		}

		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderTencentCloud,
		semconv.CloudPlatformTencentCloudCVM,
	}

	var errs []error
	for i, key := range metadataKeys {
		if values[i] == "" {
			errs = append(errs, fmt.Errorf("%s: not present in metadata", key.path))
			continue
		}
		attrs = append(attrs, key.attr(values[i]))
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
