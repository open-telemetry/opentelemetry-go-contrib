// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	linodemeta "github.com/linode/go-metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	defaultBaseURL = "http://169.254.169.254"
	tokenPath      = "/v1/token"

	// tokenTTLSeconds is the requested lifetime of a metadata token. A token is
	// consumed by the instance request issued immediately after it, so it is
	// deliberately short lived and never reused across detections.
	tokenTTLSeconds = 60

	maxResponseSize = 1 << 20
	requestTimeout  = 2 * time.Second
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

// ResourceDetector collects resource information of Akamai Connected Cloud
// compute instances.
type ResourceDetector struct {
	baseURL string
	cfg     config
	client  *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Akamai Connected Cloud compute instances.
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
		baseURL: defaultBaseURL,
		cfg:     cfg,
		client: &http.Client{
			Timeout:   requestTimeout,
			Transport: statusCheckTransport{base: transport},
		},
	}
}

// statusError reports a non-2xx response from the metadata service.
type statusError struct {
	StatusCode int
	Path       string
	Body       string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("metadata request to %s returned status %d: %s", e.Path, e.StatusCode, e.Body)
}

// statusCheckTransport converts non-2xx responses into a [statusError].
//
// The instance request is issued by github.com/linode/go-metadata, which only
// reports an error when the response body happens to decode as its own JSON
// error document. A plain-text 401 or 500 is otherwise reported as a
// successful, zero-valued instance document. Rejecting non-2xx responses at the
// transport covers both that request and the token request this package issues
// itself, and keeps the status code available to the caller.
type statusCheckTransport struct {
	base http.RoundTripper
}

func (t statusCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	_ = resp.Body.Close()
	return nil, &statusError{
		StatusCode: resp.StatusCode,
		Path:       req.URL.Path,
		Body:       string(body),
	}
}

// token mints a metadata token, which doubles as the availability probe: the
// token endpoint is the first thing the metadata service is asked for. The
// returned boolean reports whether the process appears to be running on an
// Akamai Connected Cloud instance. It is false when the metadata service cannot
// be reached, or when something other than the Akamai metadata service answered
// the link-local address.
func (d *ResourceDetector) token(ctx context.Context) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, d.baseURL+tokenPath, http.NoBody)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Metadata-Token-Expiry-Seconds", strconv.Itoa(tokenTTLSeconds))
	// The token endpoint is content negotiated. It answers with the bare token
	// as text/plain by default; this Accept header selects a JSON array holding
	// the single token string instead.
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		var serr *statusError
		if !errors.As(err, &serr) {
			// The metadata service is unreachable: not running on Akamai.
			return "", false, err
		}
		// A client error means the link-local address was answered by something
		// that is not the Akamai metadata service. Any other status is a
		// failure of the metadata service itself.
		onAkamai := serr.StatusCode < 400 || serr.StatusCode > 499
		return "", onAkamai, err
	}
	defer resp.Body.Close()

	var tokens []string
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&tokens); err != nil {
		return "", true, fmt.Errorf("failed to decode metadata token response: %w", err)
	}
	if len(tokens) == 0 || tokens[0] == "" {
		return "", true, errors.New("metadata token response contained no token")
	}
	return tokens[0], true, nil
}

// instance fetches the instance metadata document using an already minted token.
func (d *ResourceDetector) instance(ctx context.Context, token string) (*linodemeta.InstanceData, error) {
	cli, err := linodemeta.NewClient(ctx,
		linodemeta.ClientWithHTTPClient(d.client),
		// Tokens are managed by this package rather than by the client. Managed
		// tokens would mint one eagerly during construction, hiding the status
		// code that tells "not running on Akamai" apart from a metadata service
		// failure, and would mint another before every subsequent request.
		linodemeta.ClientWithoutManagedToken(),
		linodemeta.ClientWithToken(token),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata client: %w", err)
	}
	// The equivalent ClientWithBaseURL option cannot be used: NewClient applies
	// it before the underlying HTTP client exists and panics. Setting the base
	// URL after construction is safe because disabling managed tokens leaves
	// construction free of requests.
	cli.SetBaseURL(d.baseURL)

	inst, err := cli.GetInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance metadata: %w", err)
	}
	if inst == nil {
		return nil, errors.New("instance metadata response was empty")
	}
	return inst, nil
}

// Detect detects resource attributes of the Akamai Connected Cloud compute
// instance the process is running on. It returns an empty resource and no error
// when not running on an Akamai instance, and an error when the metadata
// service is reachable but does not return usable metadata. If the process is
// running on an Akamai instance but some attributes cannot be retrieved, a
// partial resource is returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	token, onAkamai, err := d.token(ctx)
	if err != nil {
		if !onAkamai {
			return resource.Empty(), nil
		}
		return nil, err
	}

	// The token request succeeded, so this is an Akamai instance. A failure
	// from here on is a metadata service failure, not a foreign host.
	inst, err := d.instance(ctx, token)
	if err != nil {
		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAkamaiCloud,
		semconv.CloudPlatformAkamaiCloudCompute,
	}

	var errs []error

	if inst.AccountEUUID == "" {
		errs = append(errs, errors.New("account ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudAccountID(inst.AccountEUUID))
	}

	if inst.Region == "" {
		errs = append(errs, errors.New("region: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.CloudRegion(inst.Region))
	}

	// host.id is the numeric instance ID rather than host_uuid, matching what
	// the collector's Akamai detector reports for the same instance.
	if inst.ID == 0 {
		errs = append(errs, errors.New("instance ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostID(strconv.Itoa(inst.ID)))
	}

	if inst.Label == "" {
		errs = append(errs, errors.New("hostname: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostName(inst.Label))
	}

	if inst.Type == "" {
		errs = append(errs, errors.New("instance type: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostType(inst.Type))
	}

	if inst.Image.ID == "" {
		errs = append(errs, errors.New("image ID: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostImageID(inst.Image.ID))
	}

	if inst.Image.Label == "" {
		errs = append(errs, errors.New("image name: not present in metadata"))
	} else {
		attrs = append(attrs, semconv.HostImageName(inst.Image.Label))
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
