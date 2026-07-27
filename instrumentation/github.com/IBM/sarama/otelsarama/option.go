// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"sync/atomic"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/IBM/sarama/otelsarama"

type config struct {
	TracerProvider trace.TracerProvider
	Propagators    propagation.TextMapPropagator
	Tracer         trace.Tracer

	// clusterID is populated asynchronously when WithClient is provided.
	// It is safe to read with atomic.Pointer.Load; nil means not yet resolved.
	clusterID atomic.Pointer[string]
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		TracerProvider: otel.GetTracerProvider(),
		Propagators:    otel.GetTextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	cfg.Tracer = cfg.TracerProvider.Tracer(
		defaultTracerName,
		trace.WithInstrumentationVersion(Version),
	)
	return cfg
}

// Option applies configuration to the otelsarama instrumentation.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) { f(cfg) }

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithPropagators specifies propagators to use for extracting and injecting
// trace context. If none are specified, the global propagators are used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		cfg.Propagators = propagators
	})
}

// WithClient specifies a sarama client to use for resolving the Kafka cluster
// ID in the background. The cluster ID is fetched once and then attached to
// all subsequent spans as messaging.kafka.cluster.id. If not provided, the
// cluster ID attribute is omitted from spans.
func WithClient(client sarama.Client) Option {
	return optionFunc(func(cfg *config) {
		go fetchClusterID(cfg, client)
	})
}

// fetchClusterID calls GetMetadata on the first available broker and stores
// the cluster ID atomically. It runs in a background goroutine and never
// blocks the instrumented code path.
func fetchClusterID(cfg *config, client sarama.Client) {
	brokers := client.Brokers()
	if len(brokers) == 0 {
		return
	}
	resp, err := brokers[0].GetMetadata(&sarama.MetadataRequest{Version: 4})
	if err != nil || resp == nil || resp.ClusterID == nil || *resp.ClusterID == "" {
		return
	}
	id := *resp.ClusterID
	cfg.clusterID.Store(&id)
}
