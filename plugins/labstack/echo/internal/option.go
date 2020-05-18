package internal

import (
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

type Config struct {
	Service     string
	Tracer      oteltrace.Tracer
	Propagators otelpropagation.Propagators
}

type Option func(*Config)

func WithTracer(tracer oteltrace.Tracer) Option {
	return func(conf *Config) {
		conf.Tracer = tracer
	}
}

func WithPropagators(propagators otelpropagation.Propagators) Option {
	return func(conf *Config) {
		conf.Propagators = propagators
	}
}
