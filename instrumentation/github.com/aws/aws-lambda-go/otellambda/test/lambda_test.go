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
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/codes"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Test Error: ", 0)

type mockIDGenerator struct {
	sync.Mutex
	traceCount int
	spanCount  int
}

func (m *mockIDGenerator) NewIDs(_ context.Context) (trace.TraceID, trace.SpanID) {
	m.Lock()
	defer m.Unlock()
	m.traceCount++
	m.spanCount++
	return [16]byte{byte(m.traceCount)}, [8]byte{byte(m.spanCount)}
}

func (m *mockIDGenerator) NewSpanID(_ context.Context, _ trace.TraceID) trace.SpanID {
	m.Lock()
	defer m.Unlock()
	m.spanCount++
	return [8]byte{byte(m.spanCount)}
}

var _ sdktrace.IDGenerator = &mockIDGenerator{}

type emptyHandler struct{}

func (h emptyHandler) Invoke(_ context.Context, _ []byte) ([]byte, error) {
	return nil, nil
}

type returnErrorHandler struct{}

func (h returnErrorHandler) Invoke(_ context.Context, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("this is an error")
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
		sdktrace.WithIDGenerator(&mockIDGenerator{}),
		sdktrace.WithResource(res),
	)

	return tp, exp
}

func setEnvVars(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "testFunction")
	t.Setenv("AWS_REGION", "us-texas-1")
	t.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	t.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "128")
	t.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
}

// Vars for Tracing and TracingWithFlusher Tests.
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
		propagation.HeaderCarrier{
			"X-Amzn-Trace-Id": []string{"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1"},
		})
	expectedTraceID, _ = trace.TraceIDFromHex("5759e988bd862e3fe1be46a994272793")
	expectedSpanStub   = tracetest.SpanStub{
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
		Attributes: []attribute.KeyValue{
			attribute.String("faas.invocation_id", "123"),
			attribute.String("aws.lambda.invoked_arn", "arn:partition:service:region:account-id:resource-type:resource-id"),
			attribute.String("cloud.account.id", "account-id"),
		},
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
			attribute.String("faas.version", "$LATEST"),
			attribute.String("faas.instance", "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"),
			attribute.Int("faas.max_memory", 128)),
		InstrumentationLibrary: instrumentation.Library{Name: "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda", Version: otellambda.Version()},
	}
)

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

func TestInstrumentHandlerTracing(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	customerHandler := func() (string, error) {
		return "hello world", nil
	}

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.InstrumentHandler(customerHandler, otellambda.WithTracerProvider(tp))
	wrappedCallable := reflect.ValueOf(wrapped)
	resp := wrappedCallable.Call([]reflect.Value{reflect.ValueOf(mockContext)})
	assert.Len(t, resp, 2)
	assert.Equal(t, "hello world", resp[0].Interface())
	assert.Nil(t, resp[1].Interface())

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)
}

func TestWrapHandlerTracing(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.WrapHandler(emptyHandler{}, otellambda.WithTracerProvider(tp))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.NoError(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)
}

type mockFlusher struct {
	flushCount int
}

func (mf *mockFlusher) ForceFlush(context.Context) error {
	mf.flushCount++
	return nil
}

var _ otellambda.Flusher = &mockFlusher{}

func TestInstrumentHandlerTracingWithFlusher(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	customerHandler := func() (string, error) {
		return "hello world", nil
	}

	flusher := mockFlusher{}
	wrapped := otellambda.InstrumentHandler(customerHandler, otellambda.WithTracerProvider(tp), otellambda.WithFlusher(&flusher))
	wrappedCallable := reflect.ValueOf(wrapped)
	resp := wrappedCallable.Call([]reflect.Value{reflect.ValueOf(mockContext)})
	assert.Len(t, resp, 2)
	assert.Equal(t, "hello world", resp[0].Interface())
	assert.Nil(t, resp[1].Interface())

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)

	assert.Equal(t, 1, flusher.flushCount)
}

func TestWrapHandlerTracingWithFlusher(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	flusher := mockFlusher{}
	wrapped := otellambda.WrapHandler(emptyHandler{}, otellambda.WithTracerProvider(tp), otellambda.WithFlusher(&flusher))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.NoError(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)

	assert.Equal(t, 1, flusher.flushCount)
}

func TestWrapHandlerTracingWithRecordErrorFalse(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	wrapped := otellambda.WrapHandler(returnErrorHandler{}, otellambda.WithTracerProvider(tp), otellambda.WithRecordError(false))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.Error(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assert.Len(t, stub.Events, 0)
}

func TestWrapHandlerTracingWithRecordErrorTrue(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	wrapped := otellambda.WrapHandler(returnErrorHandler{}, otellambda.WithTracerProvider(tp), otellambda.WithRecordError(true))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.Error(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assert.Len(t, stub.Events, 1)
	event := stub.Events[0]
	assert.Equal(t, "exception", event.Name)
}

func TestWrapHandlerTracingWithSetStatusFalse(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	wrapped := otellambda.WrapHandler(returnErrorHandler{}, otellambda.WithTracerProvider(tp), otellambda.WithSetStatus(false))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.Error(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assert.Equal(t, sdktrace.Status{Code: codes.Unset, Description: ""}, stub.Status)
}

func TestWrapHandlerTracingWithSetStatusTrue(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	wrapped := otellambda.WrapHandler(returnErrorHandler{}, otellambda.WithTracerProvider(tp), otellambda.WithSetStatus(true))
	_, err := wrapped.Invoke(mockContext, []byte{})
	assert.Error(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assert.Equal(t, sdktrace.Status{Code: codes.Error, Description: "this is an error"}, stub.Status)
}

const mockPropagatorKey = "Mockkey"

type mockPropagator struct{}

func (prop mockPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// extract tracing information
	if header := carrier.Get(mockPropagatorKey); header != "" {
		scc := trace.SpanContextConfig{}
		splitHeaderVal := strings.Split(header, ":")
		var err error
		scc.TraceID, err = trace.TraceIDFromHex(splitHeaderVal[0])
		if err != nil {
			errorLogger.Println("Failed to create trace id from hex: ", err)
		}
		scc.SpanID, err = trace.SpanIDFromHex(splitHeaderVal[1])
		if err != nil {
			errorLogger.Println("Failed to create span id from hex: ", err)
		}
		isTraced, err := strconv.Atoi(splitHeaderVal[1])
		if err != nil {
			errorLogger.Println("Failed to convert trace flag to int: ", err)
		}
		scc.TraceFlags = scc.TraceFlags.WithSampled(isTraced != 0)
		sc := trace.NewSpanContext(scc)
		return trace.ContextWithRemoteSpanContext(ctx, sc)
	}
	return ctx
}

func (prop mockPropagator) Inject(context.Context, propagation.TextMapCarrier) {
	// not needed other than to satisfy interface
}

func (prop mockPropagator) Fields() []string {
	// not needed other than to satisfy interface
	return []string{}
}

type mockRequest struct {
	Headers map[string]string
}

// Vars for mockPropagator Tests.
var (
	mockPropagatorTestsTraceIDHex = "12345678901234567890123456789012"
	mockPropagatorTestsSpanIDHex  = "1234567890123456"
	mockPropagatorTestsSampled    = "1"
	mockPropagatorTestsHeader     = mockPropagatorTestsTraceIDHex + ":" + mockPropagatorTestsSpanIDHex + ":" + mockPropagatorTestsSampled
	mockPropagatorTestsEvent      = mockRequest{Headers: map[string]string{mockPropagatorKey: mockPropagatorTestsHeader}}

	mockPropagatorTestsContext = mockPropagator{}.Extract(lambdacontext.NewContext(context.TODO(), &mockLambdaContext),
		propagation.HeaderCarrier{mockPropagatorKey: []string{mockPropagatorTestsHeader}})

	mockPropagatorTestsExpectedTraceID, _ = trace.TraceIDFromHex(mockPropagatorTestsTraceIDHex)
	mockPropagatorTestsExpectedSpanStub   = tracetest.SpanStub{
		Name: "testFunction",
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    mockPropagatorTestsExpectedTraceID,
			SpanID:     trace.SpanID{1},
			TraceFlags: 1,
			TraceState: trace.TraceState{},
			Remote:     false,
		}),
		Parent:    trace.SpanContextFromContext(mockPropagatorTestsContext),
		SpanKind:  trace.SpanKindServer,
		StartTime: time.Time{},
		EndTime:   time.Time{},
		Attributes: []attribute.KeyValue{
			attribute.String("faas.invocation_id", "123"),
			attribute.String("aws.lambda.invoked_arn", "arn:partition:service:region:account-id:resource-type:resource-id"),
			attribute.String("cloud.account.id", "account-id"),
		},
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
			attribute.String("faas.version", "$LATEST"),
			attribute.String("faas.instance", "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"),
			attribute.Int("faas.max_memory", 128)),
		InstrumentationLibrary: instrumentation.Library{Name: "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda", Version: otellambda.Version()},
	}
)

func mockRequestCarrier(eventJSON []byte) propagation.TextMapCarrier {
	var event mockRequest
	err := json.Unmarshal(eventJSON, &event)
	if err != nil {
		fmt.Println("event type: ", reflect.TypeOf(event))
		panic("mockRequestCarrier only supports events of type mockRequest")
	}
	return propagation.HeaderCarrier{mockPropagatorKey: []string{event.Headers[mockPropagatorKey]}}
}

func TestInstrumentHandlerTracingWithMockPropagator(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	customerHandler := func(event mockRequest) (string, error) {
		return "hello world", nil
	}

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.InstrumentHandler(customerHandler,
		otellambda.WithTracerProvider(tp),
		otellambda.WithPropagator(mockPropagator{}),
		otellambda.WithEventToCarrier(mockRequestCarrier))

	wrappedCallable := reflect.ValueOf(wrapped)
	resp := wrappedCallable.Call([]reflect.Value{reflect.ValueOf(mockPropagatorTestsContext), reflect.ValueOf(mockPropagatorTestsEvent)})
	assert.Len(t, resp, 2)
	assert.Equal(t, "hello world", resp[0].Interface())
	assert.Nil(t, resp[1].Interface())

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, mockPropagatorTestsExpectedSpanStub, stub)
}

func TestWrapHandlerTracingWithMockPropagator(t *testing.T) {
	setEnvVars(t)
	tp, memExporter := initMockTracerProvider()

	// No flusher needed as SimpleSpanProcessor is synchronous
	wrapped := otellambda.WrapHandler(emptyHandler{},
		otellambda.WithTracerProvider(tp),
		otellambda.WithPropagator(mockPropagator{}),
		otellambda.WithEventToCarrier(mockRequestCarrier))

	payload, _ := json.Marshal(mockPropagatorTestsEvent)
	_, err := wrapped.Invoke(mockPropagatorTestsContext, payload)
	assert.NoError(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, mockPropagatorTestsExpectedSpanStub, stub)
}
