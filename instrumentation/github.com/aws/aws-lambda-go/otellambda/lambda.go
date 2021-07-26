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

package otellambda

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Error: ", 0)

type Flusher interface {
	ForceFlush(context.Context) error
}

type noopFlusher struct{}

func (*noopFlusher) ForceFlush(context.Context) error { return nil }

// Compile time check our noopFlusher implements FLusher
var _ Flusher = &noopFlusher{}

type InstrumentationOption func(o *InstrumentationOptions)

type InstrumentationOptions struct {
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
}

var configuration InstrumentationOptions
var resourceAttributesToAddAsSpanAttributes []attribute.KeyValue
var tracer trace.Tracer

// basic implementation of TextMapCarrier
// which wraps the default map type
type mapCarrier map[string]string

// Get returns the value associated with the passed key.
func (mc mapCarrier) Get(key string) string {
	return mc[key]
}

// Set stores the key-value pair.
func (mc mapCarrier) Set(key string, value string) {
	mc[key] = value
}

// Keys lists the keys stored in this carrier.
func (mc mapCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range mc {
		keys = append(keys, k)
	}
	return keys
}

// Compile time check our mapCarrier implements propagation.TextMapCarrier
var _ propagation.TextMapCarrier = mapCarrier{}

// Logic to start OTel Tracing
func tracingBegin(ctx context.Context) (context.Context, trace.Span) {
	// Add trace id to context
	xrayTraceId := os.Getenv("_X_AMZN_TRACE_ID")
	mc := mapCarrier{}
	mc.Set("X-Amzn-Trace-Id", xrayTraceId)
	propagator := xray.Propagator{}
	ctx = propagator.Extract(ctx, mc)

	var span trace.Span
	spanName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	var attributes []attribute.KeyValue
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		errorLogger.Println("failed to load lambda context from context, ensure tracing enabled in Lambda")
	}
	if lc != nil {
		ctxRequestID := lc.AwsRequestID
		attributes = append(attributes, semconv.FaaSExecutionKey.String(ctxRequestID))

		// Some resource attrs added as span attrs because lambda
		// resource detectors are created before a lambda
		// invocation and therefore lack lambdacontext.
		// Create these attrs upon first invocation
		if resourceAttributesToAddAsSpanAttributes == nil {
			ctxFunctionArn := lc.InvokedFunctionArn
			attributes = append(attributes, semconv.FaaSIDKey.String(ctxFunctionArn))
			arnParts := strings.Split(ctxFunctionArn, ":")
			if len(arnParts) >= 5 {
				attributes = append(attributes, semconv.CloudAccountIDKey.String(arnParts[4]))
			}
		}
		attributes = append(attributes, resourceAttributesToAddAsSpanAttributes...)
	}

	ctx, span = tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attributes...))

	return ctx, span
}

// Logic to wrap up OTel Tracing
func tracingEnd(ctx context.Context, span trace.Span) {
	span.End()

	// force flush any tracing data since lambda may freeze
	err := configuration.Flusher.ForceFlush(ctx)
	if err != nil {
		errorLogger.Println("failed to force a flush, lambda may freeze before instrumentation exported: ", err)
	}
}

func WithTracerProvider(tracerProvider trace.TracerProvider) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.TracerProvider = tracerProvider
	}
}

func WithFlusher(flusher Flusher) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.Flusher = flusher
	}
}
