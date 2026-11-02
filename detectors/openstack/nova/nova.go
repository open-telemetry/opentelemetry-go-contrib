// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nova

import (
	"context"
	"encoding/json"
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
	// defaultEndpoint is the Nova instance metadata document.
	defaultEndpoint = "http://169.254.169.254/openstack/latest/meta_data.json"
	// defaultEC2Endpoint is the EC2 compatible endpoint reporting the flavor
	// of the instance. It is not served by every OpenStack deployment.
	defaultEC2Endpoint = "http://169.254.169.254/latest/meta-data/instance-type"
)

// metaPrefix namespaces the Nova instance metadata keys emitted as attributes.
const metaPrefix = "openstack.nova.meta."

// OpenStack has no cloud.provider or cloud.platform value assigned in the
// semantic conventions, so the values reported by the OpenStack detector of the
// OpenTelemetry Collector are used.
var (
	cloudProviderOpenStack     = semconv.CloudProviderKey.String("openstack")
	cloudPlatformOpenStackNova = semconv.CloudPlatformKey.String("openstack_nova")
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// metadataResponse is the JSON response from the Nova instance metadata
// service.
type metadataResponse struct {
	AvailabilityZone string            `json:"availability_zone"`
	Hostname         string            `json:"hostname"`
	Meta             map[string]string `json:"meta"`
	ProjectID        string            `json:"project_id"`
	UUID             string            `json:"uuid"`
}

type config struct {
	metaKeyFilter func(key string) bool
	filter        attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithMetaKeyFilter emits an openstack.nova.meta.<key> attribute for every
// instance metadata entry whose key satisfies filter. The filter receives the
// raw metadata key, without the "openstack.nova.meta." prefix. Without this
// option no instance metadata entries are emitted. For regexp based selection
// pass the MatchString method of a compiled [regexp.Regexp].
//
// Meta attributes are subject to [WithAttributeFilter] like any other
// attribute.
func WithMetaKeyFilter(filter func(key string) bool) Option {
	return optionFunc(func(c *config) { c.metaKeyFilter = filter })
}

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information of OpenStack Nova compute
// instances.
type ResourceDetector struct {
	endpoint    string
	ec2Endpoint string
	cfg         config
	client      *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on OpenStack Nova compute instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (169.254.169.254) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic.
	transport := &http.Transport{Proxy: nil}
	return &ResourceDetector{
		endpoint:    defaultEndpoint,
		ec2Endpoint: defaultEC2Endpoint,
		cfg:         cfg,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// get performs a GET request against url and returns the response body.
func (d *ResourceDetector) get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, 0, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// fetchMetadata queries the Nova instance metadata endpoint. The returned
// boolean reports whether the process appears to be running on an OpenStack
// Nova instance: it is false when the metadata service cannot be reached or
// when something other than the Nova metadata service answered the request.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*metadataResponse, bool, error) {
	body, status, err := d.get(ctx, d.endpoint)
	if err != nil {
		// A client error means the link-local address was answered by
		// something that is not the Nova metadata service. A transport error
		// means nothing answered at all. Any other status is a failure of the
		// metadata service itself.
		onNova := status != 0 && (status < 400 || status > 499)
		return nil, onNova, err
	}

	var meta metadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, true, err
	}
	return &meta, true, nil
}

// instanceType queries the EC2 compatible metadata endpoint for the flavor of
// the instance.
func (d *ResourceDetector) instanceType(ctx context.Context) (string, error) {
	body, _, err := d.get(ctx, d.ec2Endpoint)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// Detect detects resource attributes of the OpenStack Nova instance the
// process is running on. It returns an empty resource and no error when not
// running on a Nova instance, and an error when the metadata service is
// reachable but does not return usable metadata. If the process is running on
// a Nova instance but some attributes cannot be retrieved, a partial resource
// is returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	meta, onNova, err := d.fetchMetadata(ctx)
	if err != nil {
		if !onNova {
			return resource.Empty(), nil
		}

		return nil, err
	}

	attrs := []attribute.KeyValue{
		cloudProviderOpenStack,
		cloudPlatformOpenStackNova,
	}

	var errs []error

	for _, a := range []struct {
		field string
		value string
		attr  func(string) attribute.KeyValue
	}{
		{"project_id", meta.ProjectID, semconv.CloudAccountID},
		{"availability_zone", meta.AvailabilityZone, semconv.CloudAvailabilityZone},
		{"uuid", meta.UUID, semconv.HostID},
		{"hostname", meta.Hostname, semconv.HostName},
	} {
		if a.value == "" {
			errs = append(errs, fmt.Errorf("%s: not present in metadata", a.field))
			continue
		}
		attrs = append(attrs, a.attr(a.value))
	}

	// The EC2 compatible endpoint is not served by every OpenStack deployment,
	// so a missing host.type is not reported as a partial resource.
	if instanceType, typeErr := d.instanceType(ctx); typeErr == nil && instanceType != "" {
		attrs = append(attrs, semconv.HostType(instanceType))
	}

	// Instance metadata is namespaced, so it cannot collide with a detected
	// attribute.
	if d.cfg.metaKeyFilter != nil {
		for k, v := range meta.Meta {
			if !d.cfg.metaKeyFilter(k) {
				continue
			}
			attrs = append(attrs, attribute.String(metaPrefix+k, v))
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
