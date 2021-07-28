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

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Error: ", 0)

type instrumentor struct {
	configuration                           config
	resourceAttributesToAddAsSpanAttributes []attribute.KeyValue
	tracer                                  trace.Tracer
}

// Logic to start OTel Tracing
func (i *instrumentor) tracingBegin(ctx context.Context, eventJSON []byte) (context.Context, trace.Span) {
	// Add trace id to context
	mc := i.configuration.EventToTextMapCarrierConverter(eventJSON)
	ctx = i.configuration.Propagator.Extract(ctx, mc)

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
		if i.resourceAttributesToAddAsSpanAttributes == nil {
			ctxFunctionArn := lc.InvokedFunctionArn
			attributes = append(attributes, semconv.FaaSIDKey.String(ctxFunctionArn))
			arnParts := strings.Split(ctxFunctionArn, ":")
			if len(arnParts) >= 5 {
				attributes = append(attributes, semconv.CloudAccountIDKey.String(arnParts[4]))
			}
		}
		attributes = append(attributes, i.resourceAttributesToAddAsSpanAttributes...)
	}

	ctx, span = i.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attributes...))

	return ctx, span
}

// Logic to wrap up OTel Tracing
func (i *instrumentor) tracingEnd(ctx context.Context, span trace.Span) {
	span.End()

	// force flush any tracing data since lambda may freeze
	err := i.configuration.Flusher.ForceFlush(ctx)
	if err != nil {
		errorLogger.Println("failed to force a flush, lambda may freeze before instrumentation exported: ", err)
	}
}
