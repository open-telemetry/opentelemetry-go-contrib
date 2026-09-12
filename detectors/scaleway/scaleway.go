// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scaleway

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

const (
	// metadataPath is the path the metadata service serves the metadata
	// document of the instance from.
	metadataPath = "/conf?format=json"

	// maxBodySize bounds the response body the metadata service is trusted to
	// return.
	maxBodySize = 1 << 20

	// defaultTimeout bounds the wait for the metadata service when the context
	// passed to [ResourceDetector.Detect] carries no deadline.
	defaultTimeout = 2 * time.Second
)

// defaultEndpoints are the addresses of the Scaleway metadata service, tried in
// order.
var defaultEndpoints = []string{"http://169.254.42.42", "http://[fd00:42::42]"}

// metadataResponse is the part of the metadata document of an instance this
// detector reads.
type metadataResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Organization   string `json:"organization"`
	CommercialType string `json:"commercial_type"`
	Image          struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"image"`
	Location struct {
		ZoneID string `json:"zone_id"`
	} `json:"location"`
}

// Scaleway has no cloud.provider or cloud.platform constant in the semantic
// conventions package yet. Both values were added to the semantic conventions
// in open-telemetry/semantic-conventions#2773 and are the ones the
// OpenTelemetry Collector reports for the same instance. Replace these with the
// generated constants once they are released.
var (
	cloudProviderScaleway             = semconv.CloudProviderKey.String("scaleway_cloud")
	cloudPlatformScalewayCloudCompute = semconv.CloudPlatformKey.String("scaleway_cloud_compute")
)

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

// ResourceDetector collects resource information of Scaleway Instances.
type ResourceDetector struct {
	cfg       config
	endpoints []string
	client    *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Scaleway Instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{
		cfg:       cfg,
		endpoints: defaultEndpoints,
		// Use a transport with Proxy explicitly disabled. The metadata service
		// is reached over a link-local address that must never be contacted
		// through an HTTP(S) proxy: a proxy set for outbound traffic would
		// answer for it and make detection depend on that proxy.
		client: &http.Client{Transport: &http.Transport{Proxy: nil}},
	}
}

// fetch requests the metadata document from the metadata service at endpoint.
// The returned boolean reports whether the metadata service answered: it is
// false only when nothing responded at that address.
func (d *ResourceDetector) fetch(ctx context.Context, endpoint string) (*metadataResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+metadataPath, http.NoBody)
	if err != nil {
		return nil, false, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, true, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	var md metadataResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(&md); err != nil {
		return nil, true, fmt.Errorf("decode metadata response: %w", err)
	}
	return &md, true, nil
}

// metadata returns the metadata of the instance the process is running on. The
// addresses of the metadata service are tried in order, all under ctx. The
// returned boolean reports whether the metadata service answered at any of
// them: it is false only when the process does not appear to run on a Scaleway
// Instance.
func (d *ResourceDetector) metadata(ctx context.Context) (*metadataResponse, bool, error) {
	var (
		errs     []error
		answered bool
	)
	for _, endpoint := range d.endpoints {
		md, ok, err := d.fetch(ctx, endpoint)
		answered = answered || ok
		if err == nil {
			return md, true, nil
		}
		errs = append(errs, err)
		// An address that answered is the address of the metadata service.
		// Trying the other one would not tell us anything new.
		if ok || ctx.Err() != nil {
			break
		}
	}
	return nil, answered, errors.Join(errs...)
}

// Detect detects resource attributes of the Scaleway Instance the process is
// running on. It returns an empty resource and no error when the metadata
// service cannot be reached, which is the case when the process is not running
// on Scaleway. A metadata service that answers without serving the document is
// reported as an error. If the instance is reported but some attributes are
// missing from it, a partial resource is returned together with
// [resource.ErrPartialResource].
//
// The deadline of ctx bounds the whole detection. Without one it is bounded by
// an internal default.
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	// The caller's context is kept so that the internal bound expiring is not
	// mistaken for the caller giving up.
	fetchCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		fetchCtx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	md, answered, err := d.metadata(fetchCtx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			// The caller gave up.
			return nil, ctxErr
		}
		if answered {
			// The metadata service is there but did not serve the document.
			return nil, err
		}
		// Nothing answered at any address: not running on Scaleway.
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		cloudProviderScaleway,
		cloudPlatformScalewayCloudCompute,
	}

	var errs []error

	for _, a := range []struct {
		field string
		value string
		attr  func(string) attribute.KeyValue
	}{
		{"organization", md.Organization, semconv.CloudAccountID},
		{"location.zone_id", md.Location.ZoneID, semconv.CloudAvailabilityZone},
		{"location.zone_id (region)", zoneToRegion(md.Location.ZoneID), semconv.CloudRegion},
		{"id", md.ID, semconv.HostID},
		{"image.id", md.Image.ID, semconv.HostImageID},
		{"image.name", md.Image.Name, semconv.HostImageName},
		{"name", md.Name, semconv.HostName},
		{"commercial_type", md.CommercialType, semconv.HostType},
	} {
		if a.value == "" {
			errs = append(errs, fmt.Errorf("%s: not present in metadata", a.field))
			continue
		}
		attrs = append(attrs, a.attr(a.value))
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

// zoneToRegion returns the region a Scaleway zone belongs to, for example
// "fr-par" for the zone "fr-par-1". It returns an empty string for a zone
// carrying no region.
func zoneToRegion(zone string) string {
	if i := strings.LastIndex(zone, "-"); i > 0 {
		return zone[:i]
	}
	return ""
}
