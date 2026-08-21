// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	do "github.com/digitalocean/go-metadata"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// defaultTimeout bounds the wait for the metadata service when the context
// passed to [ResourceDetector.Detect] carries no deadline.
const defaultTimeout = 2 * time.Second

// newClient is the factory for the DigitalOcean metadata client.
// It is a package-level variable so tests can substitute a fake server.
var newClient = do.NewClient

// DigitalOcean has no cloud.provider constant in the semantic conventions
// package yet. The value is the one the OpenTelemetry Collector reports for the
// same Droplet, and was proposed for the semantic conventions in
// open-telemetry/semantic-conventions#2790. Replace this with the generated
// constant once it is released.
//
// cloud.platform is deliberately not reported: no value is standardized for
// DigitalOcean, and the Collector detector does not set one either.
var cloudProviderDigitalOcean = semconv.CloudProviderKey.String("digitalocean")

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// statusTransport records the status code of the last response it saw. The
// metadata client reports failures as a formatted string error that does not
// carry the status code, and Detect needs it to tell a client error (something
// other than the metadata service answered the link-local address) from a
// failure of the metadata service itself.
type statusTransport struct {
	base   http.RoundTripper
	status atomic.Int64
}

func (t *statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if resp != nil {
		t.status.Store(int64(resp.StatusCode))
	}
	return resp, err
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

// ResourceDetector collects resource information of DigitalOcean Droplets.
type ResourceDetector struct {
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on DigitalOcean Droplets.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// metadata queries the metadata service once and returns the whole metadata
// document, along with the status code of the response. The status code is zero
// when nothing answered.
//
// The client does not accept a [context.Context], so the request runs in its
// own goroutine and ctx only cancels the wait. That goroutine is bounded by the
// timeout of the HTTP client, but can still outlive this call.
func (*ResourceDetector) metadata(ctx context.Context) (*do.Metadata, int, error) {
	// Checked up front so an already canceled context does not race with a
	// request that may complete before the select below observes it.
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (169.254.169.254) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic. The client of the metadata package does not disable it.
	tr := &statusTransport{base: &http.Transport{Proxy: nil}}
	client := newClient(do.WithHTTPClient(&http.Client{
		Timeout:   defaultTimeout,
		Transport: tr,
	}))

	type result struct {
		md  *do.Metadata
		err error
	}

	// Buffered so the goroutine never blocks once ctx is done.
	ch := make(chan result, 1)
	go func() {
		md, err := client.Metadata()
		ch <- result{md: md, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	case r := <-ch:
		return r.md, int(tr.status.Load()), r.err
	}
}

// Detect detects resource attributes of the DigitalOcean Droplet the process is
// running on. It returns an empty resource and no error when not running on a
// Droplet, and an error when the metadata service is reachable but does not
// return usable metadata. If the process is running on a Droplet but some
// attributes cannot be retrieved, a partial resource is returned together with
// [resource.ErrPartialResource].
//
// The deadline of ctx bounds the request. Without one the request is bounded by
// an internal default.
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	md, status, err := d.metadata(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		// No response at all means the metadata service is unreachable, and a
		// client error means the link-local address was answered by something
		// that is not the DigitalOcean metadata service. Neither is a failure:
		// the process is not running on a Droplet.
		if status == 0 || (status >= 400 && status <= 499) {
			return resource.Empty(), nil
		}

		// Any other status, or a body that cannot be decoded, is a failure of
		// the metadata service itself.
		return nil, err
	}

	attrs := []attribute.KeyValue{cloudProviderDigitalOcean}

	var errs []error

	if md.DropletID == 0 {
		errs = append(errs, errors.New("droplet_id: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostID(strconv.Itoa(md.DropletID)))
	}

	// The region codes the metadata service reports are already lower case
	// (for example "nyc3"), so cloud.region is used as reported.
	for _, a := range []struct {
		field string
		value string
		attr  func(string) attribute.KeyValue
	}{
		{"hostname", md.Hostname, semconv.HostName},
		{"region", md.Region, semconv.CloudRegion},
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
