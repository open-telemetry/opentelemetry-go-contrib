// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"context"
	"fmt"
	"strings"
	"time"

	instance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// defaultTimeout bounds the wait for the metadata service when the context
// passed to [ResourceDetector.Detect] carries no deadline.
const defaultTimeout = 2 * time.Second

// metadataURLs are the addresses of the Scaleway metadata service, tried in
// order. They are set explicitly because the address the client discovers on
// its own is only resolved by probing both of them with the default HTTP
// client of the process, which the detector must not disturb.
var metadataURLs = []string{"http://169.254.42.42", "http://[fd00:42::42]"}

// newMetadataAPI is the factory for the Scaleway metadata client.
// It is a package-level variable so tests can substitute a fake server.
var newMetadataAPI = instance.NewMetadataAPI

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
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Scaleway Instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// getMetadata queries the metadata service. The client does not accept a
// [context.Context], so the request runs in its own goroutine and ctx only
// cancels the wait. The client also offers no way to bound its own request, so
// that goroutine can outlive this call.
func getMetadata(ctx context.Context, api *instance.MetadataAPI) (*instance.Metadata, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	type result struct {
		md  *instance.Metadata
		err error
	}

	// Buffered so the goroutine never blocks once ctx is done.
	ch := make(chan result, 1)
	go func() {
		md, err := api.GetMetadata()
		ch <- result{md: md, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.md, r.err
	}
}

// metadata returns the metadata of the instance the process is running on. The
// address of the metadata service is set before every request: left unset, the
// client discovers it by probing the metadata addresses with the default HTTP
// client of the process, which both changes that client's timeout and panics
// when nothing answers.
func (*ResourceDetector) metadata(ctx context.Context) (*instance.Metadata, error) {
	api := newMetadataAPI()

	// A client that already knows where to look is used as it is.
	if api.MetadataURL != nil {
		return getMetadata(ctx, api)
	}

	var firstErr error
	for _, url := range metadataURLs {
		api.MetadataURL = &url

		md, err := getMetadata(ctx, api)
		if err == nil {
			return md, nil
		}
		if firstErr == nil {
			firstErr = err
		}
		if ctx.Err() != nil {
			break
		}
	}
	return nil, firstErr
}

// Detect detects resource attributes of the Scaleway Instance the process is
// running on. It returns an empty resource and no error when the metadata
// service does not report an instance, which is also the case when the process
// is not running on Scaleway. If the instance is reported but some attributes
// are missing from it, a partial resource is returned together with
// [resource.ErrPartialResource].
//
// The deadline of ctx bounds the request. Without one the request is bounded by
// an internal default.
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	md, err := d.metadata(ctx)
	if err != nil || md == nil {
		// Only the caller canceling is an error. Every other failure means the
		// metadata service did not report an instance, and the client does not
		// report the status that would tell why.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
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
