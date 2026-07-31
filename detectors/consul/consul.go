// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/consul/api"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// defaultTimeout bounds a single request to the Consul agent.
const defaultTimeout = 2 * time.Second

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

type config struct {
	address       string
	datacenter    string
	namespace     string
	token         string
	tokenFile     string
	timeout       time.Duration
	metaKeyFilter func(key string) bool
	client        *api.Client
	filter        attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAddress sets the address of the Consul agent, for example
// "127.0.0.1:8500". By default the address resolved by the Consul client is
// used, which honors the CONSUL_HTTP_ADDR environment variable.
func WithAddress(address string) Option {
	return optionFunc(func(c *config) { c.address = address })
}

// WithDatacenter sets the datacenter to query. By default the agent's own
// datacenter is used.
func WithDatacenter(datacenter string) Option {
	return optionFunc(func(c *config) { c.datacenter = datacenter })
}

// WithNamespace sets the Consul Enterprise namespace sent with the request. By
// default the namespace resolved by the Consul client is used, which honors the
// CONSUL_NAMESPACE environment variable.
func WithNamespace(namespace string) Option {
	return optionFunc(func(c *config) { c.namespace = namespace })
}

// WithToken sets the ACL token used for the request. It is only needed when
// Consul's ACL system is enabled. By default the token resolved by the Consul
// client is used, which honors the CONSUL_HTTP_TOKEN environment variable.
func WithToken(token string) Option {
	return optionFunc(func(c *config) { c.token = token })
}

// WithTokenFile sets the path of a file containing the ACL token used for the
// request. The file is read by the Consul client. It is only needed when
// Consul's ACL system is enabled. By default the token file resolved by the
// Consul client is used, which honors the CONSUL_HTTP_TOKEN_FILE environment
// variable.
func WithTokenFile(path string) Option {
	return optionFunc(func(c *config) { c.tokenFile = path })
}

// WithTimeout sets the timeout of a single request to the Consul agent.
// Defaults to 2s. It is ignored when [WithClient] is used.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) { c.timeout = timeout })
}

// WithMetaKeyFilter emits an attribute for every Consul node meta entry whose
// key satisfies filter. The attribute key is the Consul meta key verbatim, so a
// meta key that collides with a detected attribute (for example "host.name")
// overrides it. Without this option no node meta entries are emitted. For
// regexp based selection pass the MatchString method of a compiled
// [regexp.Regexp].
func WithMetaKeyFilter(filter func(key string) bool) Option {
	return optionFunc(func(c *config) { c.metaKeyFilter = filter })
}

// WithClient sets the Consul client used to query the agent. If not set, a
// client is created during [ResourceDetector.Detect] from the other options and
// the CONSUL_* environment variables. When set, the [WithAddress],
// [WithDatacenter], [WithNamespace], [WithToken], [WithTokenFile], and
// [WithTimeout] options are ignored.
//
// Give the client a timeout. [ResourceDetector.Detect] queries from its own
// goroutine, so without one a canceled context leaves that goroutine blocked.
func WithClient(client *api.Client) Option {
	return optionFunc(func(c *config) { c.client = client })
}

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information from a Consul agent.
type ResourceDetector struct {
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes from a Consul agent.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{timeout: defaultTimeout}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// apiConfig translates the detector configuration into a Consul client
// configuration. It starts from the Consul client defaults, which resolve the
// CONSUL_* environment variables, and overrides only the explicitly configured
// values.
func (d *ResourceDetector) apiConfig() *api.Config {
	cfg := api.DefaultConfig()

	if d.cfg.address != "" {
		cfg.Address = d.cfg.address
	}
	if d.cfg.datacenter != "" {
		cfg.Datacenter = d.cfg.datacenter
	}
	if d.cfg.namespace != "" {
		cfg.Namespace = d.cfg.namespace
	}
	if d.cfg.token != "" {
		cfg.Token = d.cfg.token
	}
	if d.cfg.tokenFile != "" {
		cfg.TokenFile = d.cfg.tokenFile
	}

	return cfg
}

// newClient returns a Consul client built from the detector configuration.
func (d *ResourceDetector) newClient() (*api.Client, error) {
	cfg := d.apiConfig()

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// api.NewClient builds the HTTP client from the transport and TLS
	// configuration, so the timeout can only be applied afterwards. It assigns
	// the client back onto cfg, and the returned client holds the same pointer.
	if cfg.HttpClient != nil {
		cfg.HttpClient.Timeout = d.cfg.timeout
	}

	return client, nil
}

// self queries the agent configuration endpoint. The Consul client does not
// accept a [context.Context], so the request runs in its own goroutine and ctx
// only cancels the wait. The client timeout bounds that goroutine.
func self(ctx context.Context, client *api.Client) (map[string]map[string]any, error) {
	type result struct {
		self map[string]map[string]any
		err  error
	}

	// Buffered so the goroutine never blocks once ctx is done.
	ch := make(chan result, 1)
	go func() {
		s, err := client.Agent().Self()
		ch <- result{self: s, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.self, r.err
	}
}

// isDialFailure reports whether err is a failure to connect to the agent.
func isDialFailure(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

// Detect detects resource attributes of the Consul agent. It returns an empty
// resource and no error when no agent can be reached, and an error when the
// agent answers without a usable configuration. Missing individual attributes
// yield a partial resource with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	client := d.cfg.client
	if client == nil {
		var err error
		if client, err = d.newClient(); err != nil {
			return nil, fmt.Errorf("failed creating consul client: %w", err)
		}
	}

	agent, err := self(ctx, client)
	if err != nil {
		if isDialFailure(err) {
			// Nothing answered: no Consul agent here.
			return resource.Empty(), nil
		}
		return nil, fmt.Errorf("failed to get local agent information: %w", err)
	}

	// The key can be present with a null value, so check both.
	agentCfg, ok := agent["Config"]
	if !ok || agentCfg == nil {
		return nil, errors.New("consul agent did not report a Config section")
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	for _, a := range []struct {
		key  string
		attr func(string) attribute.KeyValue
	}{
		{"NodeName", semconv.HostName},
		{"Datacenter", semconv.CloudRegion},
		{"NodeID", semconv.HostID},
	} {
		v, ok := agentCfg[a.key].(string)
		if !ok || v == "" {
			errs = append(errs, fmt.Errorf("%s: not present in agent configuration", a.key))
			continue
		}
		attrs = append(attrs, a.attr(v))
	}

	// Node meta is appended last so a meta key that collides with a detected
	// attribute overrides it, matching the collector's Consul detector.
	if d.cfg.metaKeyFilter != nil {
		for k, v := range agent["Meta"] {
			s, ok := v.(string)
			if !ok || !d.cfg.metaKeyFilter(k) {
				continue
			}
			attrs = append(attrs, attribute.String(k, s))
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
