// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xrayconfig

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	v1common "go.opentelemetry.io/proto/otlp/common/v1"
	v1resource "go.opentelemetry.io/proto/otlp/resource/v1"
	v1trace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestEventToCarrier(t *testing.T) {
	t.Setenv("_X_AMZN_TRACE_ID", "traceID")
	carrier := xrayEventToCarrier([]byte{})

	assert.Equal(t, "traceID", carrier.Get("X-Amzn-Trace-Id"))
}

func TestEventToCarrierWithPropagator(t *testing.T) {
	t.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
	carrier := xrayEventToCarrier([]byte{})
	ctx := xray.Propagator{}.Extract(context.Background(), carrier)

	expectedTraceID, _ := trace.TraceIDFromHex("5759e988bd862e3fe1be46a994272793")
	expectedSpanID, _ := trace.SpanIDFromHex("53995c3f42cd8ad8")
	expectedCtx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    expectedTraceID,
		SpanID:     expectedSpanID,
		TraceFlags: trace.FlagsSampled,
		TraceState: trace.TraceState{},
		Remote:     true,
	}))

	assert.Equal(t, expectedCtx, ctx)
}

func setEnvVars(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "testFunction")
	t.Setenv("AWS_REGION", "us-texas-1")
	t.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	t.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "128")
	t.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")

	// fix issue: "The requested service provider could not be loaded or initialized."
	// Guess: The env for Windows in GitHub action is incomplete
	if runtime.GOOS == "windows" && os.Getenv("SYSTEMROOT") == "" {
		t.Setenv("SYSTEMROOT", `C:\Windows`)
	}
}

// Vars for end to end testing.
var (
	mockLambdaContext = lambdacontext.LambdaContext{
		AwsRequestID:       "123",
		InvokedFunctionArn: "arn:partition:service:region:account-id:resource-type:resource-id",
		Identity:           lambdacontext.CognitoIdentity{},
		ClientContext:      lambdacontext.ClientContext{},
	}
	mockContext = xray.Propagator{}.Extract(lambdacontext.NewContext(context.Background(), &mockLambdaContext),
		propagation.HeaderCarrier{
			"X-Amzn-Trace-Id": []string{"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1"},
		})

	expectedSpans = v1trace.ScopeSpans{
		Scope: &v1common.InstrumentationScope{Name: "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda", Version: otellambda.Version()},
		Spans: []*v1trace.Span{{
			TraceId:           []byte{0x57, 0x59, 0xe9, 0x88, 0xbd, 0x86, 0x2e, 0x3f, 0xe1, 0xbe, 0x46, 0xa9, 0x94, 0x27, 0x27, 0x93},
			SpanId:            nil,
			TraceState:        "",
			ParentSpanId:      []byte{0x53, 0x99, 0x5c, 0x3f, 0x42, 0xcd, 0x8a, 0xd8},
			Name:              "testFunction",
			Kind:              v1trace.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: 0,
			EndTimeUnixNano:   0,
			Attributes: []*v1common.KeyValue{
				{Key: "faas.invocation_id", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "123"}}},
				{Key: "aws.lambda.invoked_arn", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "arn:partition:service:region:account-id:resource-type:resource-id"}}},
				{Key: "cloud.account.id", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "account-id"}}},
			},
			DroppedAttributesCount: 0,
			Events:                 nil,
			DroppedEventsCount:     0,
			Links:                  nil,
			DroppedLinksCount:      0,
			Status:                 &v1trace.Status{Code: v1trace.Status_STATUS_CODE_UNSET},
		}},
		SchemaUrl: "",
	}

	expectedSpanResource = v1resource.Resource{
		Attributes: []*v1common.KeyValue{
			{Key: "cloud.provider", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "aws"}}},
			{Key: "cloud.region", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "us-texas-1"}}},
			{Key: "faas.instance", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"}}},
			{Key: "faas.max_memory", Value: &v1common.AnyValue{Value: &v1common.AnyValue_IntValue{IntValue: 128}}},
			{Key: "faas.name", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "testFunction"}}},
			{Key: "faas.version", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "$LATEST"}}},
		},
		DroppedAttributesCount: 0,
	}

	expectedResourceSpans = v1trace.ResourceSpans{
		Resource:   &expectedSpanResource,
		ScopeSpans: []*v1trace.ScopeSpans{&expectedSpans},
		SchemaUrl:  "",
	}
)

func assertResourceEquals(t *testing.T, expected *v1resource.Resource, actual *v1resource.Resource) {
	assert.Len(t, actual.Attributes, 6)
	assert.Equal(t, expected.Attributes[0].String(), actual.Attributes[0].String())
	assert.Equal(t, expected.Attributes[1].String(), actual.Attributes[1].String())
	assert.Equal(t, expected.Attributes[2].String(), actual.Attributes[2].String())
	assert.Equal(t, expected.Attributes[3].String(), actual.Attributes[3].String())
	assert.Equal(t, expected.Attributes[4].String(), actual.Attributes[4].String())
	assert.Equal(t, expected.Attributes[5].String(), actual.Attributes[5].String())
	assert.Equal(t, expected.DroppedAttributesCount, actual.DroppedAttributesCount)
}

// ignore timestamps and SpanID since time is obviously variable,
// and SpanID is randomized when using xray IDGenerator.
func assertSpanEqualsIgnoreTimeAndSpanID(t *testing.T, expected *v1trace.ResourceSpans, actual *v1trace.ResourceSpans) {
	assert.Equal(t, expected.ScopeSpans[0].Scope, actual.ScopeSpans[0].Scope)

	actualSpan := actual.ScopeSpans[0].Spans[0]
	expectedSpan := expected.ScopeSpans[0].Spans[0]
	assert.Equal(t, expectedSpan.Name, actualSpan.Name)
	assert.Equal(t, expectedSpan.ParentSpanId, actualSpan.ParentSpanId)
	assert.Equal(t, expectedSpan.Kind, actualSpan.Kind)
	assert.Equal(t, expectedSpan.Attributes, actualSpan.Attributes)
	assert.Equal(t, expectedSpan.Events, actualSpan.Events)
	assert.Equal(t, expectedSpan.Links, actualSpan.Links)
	assert.Equal(t, expectedSpan.Status, actualSpan.Status)
	assert.Equal(t, expectedSpan.DroppedAttributesCount, actualSpan.DroppedAttributesCount)
	assert.Equal(t, expectedSpan.DroppedEventsCount, actualSpan.DroppedEventsCount)
	assert.Equal(t, expectedSpan.DroppedLinksCount, actualSpan.DroppedLinksCount)

	assertResourceEquals(t, expected.Resource, actual.Resource)
}

func TestWrapEndToEnd(t *testing.T) {
	setEnvVars(t)

	ctx := context.Background()
	tp, err := NewTracerProvider(ctx)
	assert.NoError(t, err)

	customerHandler := func() (string, error) {
		return "hello world", nil
	}
	mockCollector := runMockCollectorAtEndpoint(t, "localhost:4317")
	defer func() {
		_ = mockCollector.Stop()
	}()
	<-time.After(5 * time.Millisecond)

	wrapped := otellambda.InstrumentHandler(customerHandler, WithRecommendedOptions(tp)...)
	wrappedCallable := reflect.ValueOf(wrapped)
	resp := wrappedCallable.Call([]reflect.Value{reflect.ValueOf(mockContext)})
	assert.Len(t, resp, 2)
	assert.Equal(t, "hello world", resp[0].Interface())
	assert.Nil(t, resp[1].Interface())

	resSpans := mockCollector.getResourceSpans()
	assert.Len(t, resSpans, 1)
	assertSpanEqualsIgnoreTimeAndSpanID(t, &expectedResourceSpans, resSpans[0])
}
