// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package classic

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
	defaultEndpoint = "https://api.service.softlayer.com/rest/v3.1/SoftLayer_Resource_Metadata"
	maxResponseSize = 1 << 20

	// TODO: Use the semantic convention constant when one is added.
	cloudPlatformIBMCloudClassic = "ibm_cloud.classic"
)

// The SoftLayer Resource Metadata service exposes one endpoint per field in its
// plain-text (.txt) form; there is no aggregate instance document to fetch
// instead. idPath doubles as the availability probe: it is the canonical
// identity endpoint, so its response decides whether this process is running on
// an IBM Cloud Classic instance at all.
const (
	idPath               = "getId.txt"
	hostnamePath         = "getHostname.txt"
	datacenterPath       = "getDatacenter.txt"
	accountIDPath        = "getAccountId.txt"
	globalIdentifierPath = "getGlobalIdentifier.txt"
)

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

// ResourceDetector queries the IBM Cloud Classic (SoftLayer) Resource Metadata
// service and emits resource attributes for the current instance.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on IBM Cloud Classic (SoftLayer) instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. Instance metadata must
	// never be routed through an HTTP(S) proxy: doing so could leak it to
	// whatever the user configured in HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic.
	transport := &http.Transport{Proxy: nil}
	return &ResourceDetector{
		endpoint: defaultEndpoint,
		cfg:      cfg,
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
		},
	}
}

// fetchResult is the outcome of a single metadata field request.
type fetchResult struct {
	value string
	err   error
	// onClassic reports whether the response looks like it came from the
	// SoftLayer metadata service. It is false when the service is unreachable
	// or answered with a client error.
	onClassic bool
}

// fetch retrieves a single plain-text metadata field.
func (d *ResourceDetector) fetch(ctx context.Context, path string) fetchResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/"+path, http.NoBody)
	if err != nil {
		return fetchResult{err: err, onClassic: true}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on IBM Cloud Classic.
		return fetchResult{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the request was answered by something that is
		// not the SoftLayer metadata service. Any other status is a failure of
		// the metadata service itself.
		onClassic := resp.StatusCode < 400 || resp.StatusCode > 499
		return fetchResult{
			err:       fmt.Errorf("%s returned status %d", path, resp.StatusCode),
			onClassic: onClassic,
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fetchResult{err: err, onClassic: true}
	}

	// The .txt form returns the value unquoted, with a trailing newline.
	return fetchResult{value: strings.TrimSpace(string(body)), onClassic: true}
}

// instanceMetadata fetches every metadata field concurrently. Unlike the
// collector implementation this port is derived from, a failing field does not
// cancel the others: each result is reported independently so a single
// unavailable field yields a partial resource instead of no resource at all.
func (d *ResourceDetector) instanceMetadata(ctx context.Context) map[string]fetchResult {
	paths := []string{
		idPath,
		hostnamePath,
		datacenterPath,
		accountIDPath,
		globalIdentifierPath,
	}

	results := make(map[string]fetchResult, len(paths))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, path := range paths {
		wg.Go(func() {
			r := d.fetch(ctx, path)
			mu.Lock()
			defer mu.Unlock()
			results[path] = r
		})
	}
	wg.Wait()

	return results
}

// Detect detects resource attributes of the IBM Cloud Classic instance the
// process is running on. It returns an empty resource and no error when not
// running on an IBM Cloud Classic instance, and an error when the metadata
// service is reachable but does not return usable metadata. If the process is
// running on an IBM Cloud Classic instance but some attributes cannot be
// retrieved, a partial resource is returned together with
// [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	results := d.instanceMetadata(ctx)

	// getId.txt is the availability probe: if it did not come from the metadata
	// service, this process is not on an IBM Cloud Classic instance.
	if probe := results[idPath]; probe.err != nil {
		if !probe.onClassic {
			return resource.Empty(), nil
		}
		return nil, probe.err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String(cloudPlatformIBMCloudClassic),
	}

	var errs []error

	// Every field the metadata service exposes is required: a missing one makes
	// the resource partial rather than being silently emitted as an empty
	// attribute.
	for _, f := range []struct {
		path string
		name string
		attr func(string) attribute.KeyValue
	}{
		{idPath, "host ID", semconv.HostID},
		{hostnamePath, "hostname", semconv.HostName},
		{datacenterPath, "datacenter", semconv.CloudAvailabilityZone},
		{accountIDPath, "account ID", semconv.CloudAccountID},
		{globalIdentifierPath, "global identifier", semconv.CloudResourceID},
	} {
		r := results[f.path]
		switch {
		case r.err != nil:
			errs = append(errs, fmt.Errorf("%s: %w", f.name, r.err))
		case r.value == "":
			errs = append(errs, fmt.Errorf("%s: not present in metadata", f.name))
		default:
			attrs = append(attrs, f.attr(r.value))
		}
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
