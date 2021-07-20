package otellambda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

func initMockTracerProvider() *tracetest.InMemoryExporter {
	ctx := context.Background()

	exp := tracetest.NewInMemoryExporter()

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(ctx)
	if err != nil {
		errorLogger.Printf("failed to detect lambda resources: %v\n", err)
		return nil
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithIDGenerator(&mockIdGenerator{}),
		sdktrace.WithResource(res),
	)

	// Set the traceprovider
	otel.SetTracerProvider(tp)

	return exp
}

func setEnvVars() {
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "testFunction")
	_ = os.Setenv("AWS_REGION", "us-texas-1")
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	_ = os.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
}

func TestLambdaHandlerSignatures(t *testing.T) {
	setEnvVars()

	// for these tests we do not care about the tracing and
	// so we will ignore it the in memory span exporter
	_ = initMockTracerProvider()

	emptyPayload := ""
	testCases := []struct {
		name     string
		handler  interface{}
		expected error
		args     []reflect.Value
	}{
		{
			name:     "nil handler",
			expected: errors.New("handler is nil"),
			handler:  nil,
			args:     []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler is not a function",
			expected: errors.New("handler kind struct is not func"),
			handler:  struct{}{},
			args:     []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler declares too many arguments",
			expected: errors.New("handlers may not take more than two arguments, but handler takes 3"),
			handler: func(n context.Context, x string, y string) error {
				return nil
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "two argument handler does not have context as first argument",
			expected: errors.New("handler takes two arguments, but the first is not Context. got string"),
			handler: func(a string, x context.Context) error {
				return nil
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler returns too many values",
			expected: errors.New("handler may not return more than two values"),
			handler: func() (error, error, error) {
				return nil, nil, nil
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler returning two values does not declare error as the second return value",
			expected: errors.New("handler returns two values, but the second does not implement error"),
			handler: func() (error, string) {
				return nil, "hello"
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler returning a single value does not implement error",
			expected: errors.New("handler returns a single value, but it does not implement error"),
			handler: func() string {
				return "hello"
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "no args or return value should not result in error",
			expected: nil,
			handler: func() {
			},
			args: []reflect.Value{reflect.ValueOf(mockContext)}, // reminder - customer takes no args but wrapped handler always takes context from lambda
		},
	}
	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("testCase[%d] %s", i, testCase.name), func(t *testing.T) {
			lambdaHandler := LambdaHandlerWrapper(testCase.handler)
			handler := reflect.ValueOf(lambdaHandler)
			resp := handler.Call(testCase.args)
			assert.Equal(t, 2, len(resp))
			assert.Equal(t, testCase.expected, resp[1].Interface())
		})
	}
}

type expected struct {
	val interface{}
	err error
}

func TestHandlerInvokes(t *testing.T) {
	setEnvVars()

	// for these tests we do not care about the tracing and
	// so we will ignore it the in memory span exporter
	_ = initMockTracerProvider()

	hello := func(s string) string {
		return fmt.Sprintf("Hello %s!", s)
	}

	testCases := []struct {
		name     string
		input    interface{}
		expected expected
		handler  interface{}
	}{
		{
			name:     "string input and return without context",
			input:    "Lambda",
			expected: expected{`"Hello Lambda!"`, nil},
			handler: func(name string) (string, error) {
				return hello(name), nil
			},
		},
		{
			name:     "string input and return with context",
			input:    "Lambda",
			expected: expected{`"Hello Lambda!"`, nil},
			handler: func(ctx context.Context, name string) (string, error) {
				return hello(name), nil
			},
		},
		{
			name:     "no input with response event and simple error",
			input:    nil,
			expected: expected{"", errors.New("bad stuff")},
			handler: func() (interface{}, error) {
				return nil, errors.New("bad stuff")
			},
		},
		{
			name:     "input with response event and simple error",
			input:    "Lambda",
			expected: expected{"", errors.New("bad stuff")},
			handler: func(e interface{}) (interface{}, error) {
				return nil, errors.New("bad stuff")
			},
		},
		{
			name:     "input and context with response event and simple error",
			input:    "Lambda",
			expected: expected{"", errors.New("bad stuff")},
			handler: func(ctx context.Context, e interface{}) (interface{}, error) {
				return nil, errors.New("bad stuff")
			},
		},
		{
			name:     "input with response event and complex error",
			input:    "Lambda",
			expected: expected{"", messages.InvokeResponse_Error{Message: "message", Type: "type"}},
			handler: func(e interface{}) (interface{}, error) {
				return nil, messages.InvokeResponse_Error{Message: "message", Type: "type"}
			},
		},
		{
			name:     "basic input struct serialization",
			input:    struct{ Custom int }{9001},
			expected: expected{`9001`, nil},
			handler: func(event struct{ Custom int }) (int, error) {
				return event.Custom, nil
			},
		},
		{
			name:     "basic output struct serialization",
			input:    9001,
			expected: expected{`{"Number":9001}`, nil},
			handler: func(event int) (struct{ Number int }, error) {
				return struct{ Number int }{event}, nil
			},
		},
	}

	// test invocation via a lambda handler
	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("lambdaHandlerTestCase[%d] %s", i, testCase.name), func(t *testing.T) {
			lambdaHandler := LambdaHandlerWrapper(testCase.handler)
			handler := reflect.ValueOf(lambdaHandler)
			handlerType := handler.Type()

			var args []reflect.Value
			args = append(args, reflect.ValueOf(mockContext))
			if handlerType.NumIn() > 1 {
				args = append(args, reflect.ValueOf(testCase.input))
			}
			response := handler.Call(args)
			assert.Equal(t, 2, len(response))
			if testCase.expected.err != nil {
				assert.Equal(t, testCase.expected.err, response[handlerType.NumOut()-1].Interface())
			} else {
				assert.Nil(t, response[handlerType.NumOut()-1].Interface())
				responseValMarshalled, _ := json.Marshal(response[0].Interface())
				assert.Equal(t, testCase.expected.val, string(responseValMarshalled))
			}
		})
	}

	// test invocation via a Handler
	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("handlerTestCase[%d] %s", i, testCase.name), func(t *testing.T) {
			handler := HandlerWrapper(lambda.NewHandler(testCase.handler))
			inputPayload, _ := json.Marshal(testCase.input)
			response, err := handler.Invoke(mockContext, inputPayload)
			if testCase.expected.err != nil {
				assert.Equal(t, testCase.expected.err, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected.val, string(response))
			}
		})
	}
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
	InstrumentationLibrary: instrumentation.Library{Name: "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"},
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
	memExporter := initMockTracerProvider()

	customerHandler := func() (string, error) {
		return "hello world", nil
	}

	wrapped := LambdaHandlerWrapper(customerHandler)
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
	memExporter := initMockTracerProvider()

	wrapped := HandlerWrapper(emptyHandler{})
	_, err := wrapped.Invoke(mockContext, nil)
	assert.NoError(t, err)

	assert.Len(t, memExporter.GetSpans(), 1)
	stub := memExporter.GetSpans()[0]
	assertStubEqualsIgnoreTime(t, expectedSpanStub, stub)
}

func BenchmarkLambdaHandlerWrapper(b *testing.B) {
	setEnvVars()
	initMockTracerProvider()

	customerHandler := func(ctx context.Context, payload int) error {
		return nil
	}
	wrapped := LambdaHandlerWrapper(customerHandler)
	wrappedCallable := reflect.ValueOf(wrapped)
	ctx := reflect.ValueOf(mockContext)
	payload := reflect.ValueOf(0)
	args := []reflect.Value{ctx, payload}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrappedCallable.Call(args)
	}
}

func BenchmarkHandlerWrapper(b *testing.B) {
	setEnvVars()
	initMockTracerProvider()

	wrapped := HandlerWrapper(emptyHandler{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrapped.Invoke(mockContext, []byte{0})
	}
}
