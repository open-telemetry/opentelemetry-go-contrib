// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// A Flusher dictates how the instrumentation will attempt to flush
// unexported spans at the end of each Lambda innovation. This is
// very important in asynchronous settings because the Lambda runtime
// may enter a 'frozen' state any time after the invocation completes.
// Should this freeze happen and spans are left unexported, there can be a
// long delay before those spans are exported.
type Flusher interface {
	ForceFlush(context.Context) error
}

type noopFlusher struct{}

func (*noopFlusher) ForceFlush(context.Context) error { return nil }

// Compile time check our noopFlusher implements Flusher.
var _ Flusher = &noopFlusher{}

// An EventToCarrier function defines how the instrumentation should
// prepare a TextMapCarrier for the configured propagator to read from. This
// extra step is necessary because Lambda does not have HTTP headers to read
// from and instead stores the headers it was invoked with (including TraceID, etc.)
// as part of the invocation event. If using the AWS XRay tracing then the
// trace information is instead stored in the Lambda environment.
type EventToCarrier func(eventJSON []byte) propagation.TextMapCarrier

func emptyEventToCarrier([]byte) propagation.TextMapCarrier {
	return propagation.HeaderCarrier{}
}

// Compile time check our emptyEventToCarrier implements EventToCarrier.
var _ EventToCarrier = emptyEventToCarrier

// Option applies a configuration option.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

type config struct {
	// TracerProvider is the TracerProvider which will be used
	// to create instrumentation spans
	// The default value of TracerProvider the global otel TracerProvider
	// returned by otel.GetTracerProvider()
	TracerProvider trace.TracerProvider

	// Flusher is the mechanism used to flush any unexported spans
	// each Lambda Invocation to avoid spans being unexported for long
	// when periods of time if Lambda freezes the execution environment
	// The default value of Flusher is a noop Flusher, using this
	// default can result in long data delays in asynchronous settings
	Flusher Flusher

	// EventToCarrier is the mechanism used to retrieve the TraceID
	// from the event or environment and generate a TextMapCarrier which
	// can then be used by a Propagator to extract the TraceID into our context
	// The default value of eventToCarrier is emptyEventToCarrier which returns
	// an empty HeaderCarrier, using this default will cause new spans to be part
	// of a new Trace and have no parent past our Lambda instrumentation span
	EventToCarrier EventToCarrier

	// Propagator is the Propagator which will be used
	// to extract Trace info into the context
	// The default value of Propagator the global otel Propagator
	// returned by otel.GetTextMapPropagator()
	Propagator propagation.TextMapPropagator

	// RecordError is used to determine if an error returned from the wrapped
	// handler should be recorded on the instrumentation-provided span
	RecordError bool

	// RecordStackTrace is used to determine if the stack trace should be
	// included when RecordError is true
	RecordStackTrace bool
}

// WithTracerProvider configures the TracerProvider used by the
// instrumentation.
//
// By default, the global TracerProvider is used.
func WithTracerProvider(tracerProvider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		c.TracerProvider = tracerProvider
	})
}

// WithFlusher sets the used flusher.
func WithFlusher(flusher Flusher) Option {
	return optionFunc(func(c *config) {
		c.Flusher = flusher
	})
}

// WithEventToCarrier sets the used EventToCarrier.
func WithEventToCarrier(eventToCarrier EventToCarrier) Option {
	return optionFunc(func(c *config) {
		c.EventToCarrier = eventToCarrier
	})
}

// WithPropagator configures the propagator used by the instrumentation.
//
// By default, the global TextMapPropagator will be used.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		c.Propagator = propagator
	})
}

// WithRecordError configures whether the instrumentation should record an
// error returned by the wrapped lambda on the instrumentation-provided span.
//
// By default, returned errors are not recorded.
func WithRecordError(recordError bool, recordStackTrace bool) Option {
	return optionFunc(func(c *config) {
		c.RecordError = recordError
		c.RecordStackTrace = recordError && recordStackTrace
	})
}
