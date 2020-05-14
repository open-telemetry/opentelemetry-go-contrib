// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.
// Copyright 2020 The OpenTelemetry Authors

package mongo

import (
	"go.opentelemetry.io/otel/api/trace"
)

// Option represents an option that can be passed to Dial.
type Option func(*config)

func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *config) {
		cfg.tracer = tracer
	}
}

// WithServiceName sets the given service name for the dialed connection.
// When the service name is not explicitly set it will be inferred based on the
// request to AWS.
func WithServiceName(name string) Option {
	return func(cfg *config) {
		cfg.serviceName = name
	}
}

type config struct {
	tracer      trace.Tracer
	serviceName string
}

func newConfig(opts ...Option) config {
	var c config

	defaultOpts := []Option{
		WithTracer(trace.NoopTracer{}),
		WithServiceName("mongo"),
	}

	for _, opt := range append(defaultOpts, opts...) {
		opt(&c)
	}

	return c
}
