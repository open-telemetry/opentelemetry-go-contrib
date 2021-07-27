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

package test

import (
	"context"
	"log"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	mockLambdaContext = lambdacontext.LambdaContext{
		AwsRequestID:       "123",
		InvokedFunctionArn: "arn:partition:service:region:account-id:resource-type:resource-id",
		Identity: lambdacontext.CognitoIdentity{
			CognitoIdentityID:     "someId",
			CognitoIdentityPoolID: "somePoolId",
		},
		ClientContext: lambdacontext.ClientContext{},
	}
	mockContext = xray.Propagator{}.Extract(lambdacontext.NewContext(context.TODO(), &mockLambdaContext),
		mapCarrier{
			"X-Amzn-Trace-Id": "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
		})
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Test Error: ", 0)

// basic implementation of TextMapCarrier
// which wraps the default map type
type mapCarrier map[string]string

// Compile time check our mapCarrier implements propagation.TextMapCarrier
var _ propagation.TextMapCarrier = mapCarrier{}

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

type mockIdGenerator struct {
	sync.Mutex
	traceCount int
	spanCount  int
}

func (m *mockIdGenerator) NewIDs(_ context.Context) (trace.TraceID, trace.SpanID) {
	m.Lock()
	defer m.Unlock()
	m.traceCount += 1
	m.spanCount += 1
	return [16]byte{byte(m.traceCount)}, [8]byte{byte(m.spanCount)}
}

func (m *mockIdGenerator) NewSpanID(_ context.Context, _ trace.TraceID) trace.SpanID {
	m.Lock()
	defer m.Unlock()
	m.spanCount += 1
	return [8]byte{byte(m.spanCount)}
}

var _ sdktrace.IDGenerator = &mockIdGenerator{}

type emptyHandler struct{}

func (h emptyHandler) Invoke(_ context.Context, _ []byte) ([]byte, error) {
	return nil, nil
}

var _ lambda.Handler = emptyHandler{}

func initMockTracerProvider() (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	ctx := context.Background()

	exp := tracetest.NewInMemoryExporter()

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(ctx)
	if err != nil {
		errorLogger.Printf("failed to detect lambda resources: %v\n", err)
		return nil, nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithIDGenerator(&mockIdGenerator{}),
		sdktrace.WithResource(res),
	)

	return tp, exp
}

func setEnvVars() {
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "testFunction")
	_ = os.Setenv("AWS_REGION", "us-texas-1")
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	_ = os.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
}

var expectedTraceID, _ = trace.TraceIDFromHex("5759e988bd862e3fe1be46a994272793")
var expectedSpanStub = tracetest.SpanStub{
	Name: "testFunction",
	SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    expectedTraceID,
		SpanID:     trace.SpanID{1},
		TraceFlags: 1,
		TraceState: trace.TraceState{},
		Remote:     false,
	}),
	Parent:    trace.SpanContextFromContext(mockContext),
	SpanKind:  trace.SpanKindServer,
	StartTime: time.Time{},
	EndTime:   time.Time{},
	Attributes: []attribute.KeyValue{attribute.String("faas.execution", "123"),
		attribute.String("faas.id", "arn:partition:service:region:account-id:resource-type:resource-id"),
		attribute.String("cloud.account.id", "account-id")},
	Events:            nil,
	Links:             nil,
	Status:            sdktrace.Status{},
	DroppedAttributes: 0,
	DroppedEvents:     0,
	DroppedLinks:      0,
	ChildSpanCount:    0,
	Resource: resource.NewWithAttributes(semconv.SchemaURL,
		attribute.String("cloud.provider", "aws"),
		attribute.String("cloud.region", "us-texas-1"),
		attribute.String("faas.name", "testFunction"),
		attribute.String("faas.version", "$LATEST")),
	InstrumentationLibrary: instrumentation.Library{Name: "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda", Version: contrib.SemVersion()},
}

func assertStubEqualsIgnoreTime(t *testing.T, expected tracetest.SpanStub, actual tracetest.SpanStub) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.SpanContext, actual.SpanContext)
	assert.Equal(t, expected.Parent, actual.Parent)
	assert.Equal(t, expected.SpanKind, actual.SpanKind)
	assert.Equal(t, expected.Attributes, actual.Attributes)
	assert.Equal(t, expected.Events, actual.Events)
	assert.Equal(t, expected.Links, actual.Links)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.DroppedAttributes, actual.DroppedAttributes)
	assert.Equal(t, expected.DroppedEvents, actual.DroppedEvents)
	assert.Equal(t, expected.DroppedLinks, actual.DroppedLinks)
	assert.Equal(t, expected.ChildSpanCount, actual.ChildSpanCount)
	assert.Equal(t, expected.Resource, actual.Resource)
	assert.Equal(t, expected.InstrumentationLibrary, actual.InstrumentationLibrary)
}

func TestLambdaHandlerWrapperTracing(t *testing.T) {
	setEnvVars()
	tp, memExporter := initMockTracerProvider()

	customerHandler := func() (string, error) {
		return "hello world", nil
	}

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.WrapHandlerFunction(customerHandler, otellambda.WithTracerProvider(tp))
	wrappedCallable := reflect.ValueOf(wrapped)
	resp := wrappedCallable.Call([]reflect.Value{reflect.ValueOf(mockContext)})
	assert.Len(t, resp, 2)
	assert.Equal(t, "hello world", resp[0].Interface())
	assert.Nil(t, resp[1].Interface())

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)
}

func TestHandlerWrapperTracing(t *testing.T) {
	setEnvVars()
	tp, memExporter := initMockTracerProvider()

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.WrapHandler(emptyHandler{}, otellambda.WithTracerProvider(tp))
	_, err := wrapped.Invoke(mockContext, nil)
	assert.NoError(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)
}
