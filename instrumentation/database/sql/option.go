package otelsql

import (
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type traceAttributes []label.KeyValue

// Config is used to configure the go-restful middleware.
type config struct {
	traceProvider   trace.Provider
	traceAttributes traceAttributes
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithTracer configures the interceptor with the specified trace provider.
func WithTraceProvider(traceProvider trace.Provider) Option {
	return func(cfg *config) {
		cfg.traceProvider = traceProvider
	}
}

// WithTracer configures the interceptor to attach the default KeyValues.
func WithTraceAttributes(traceAttributes []label.KeyValue) Option {
	return func(cfg *config) {
		cfg.traceAttributes = traceAttributes
	}
}
