// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellambda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
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
	mockContext = lambdacontext.NewContext(context.TODO(), &mockLambdaContext)
)

type emptyHandler struct{}

func (h emptyHandler) Invoke(_ context.Context, _ []byte) ([]byte, error) {
	return nil, nil
}

var _ lambda.Handler = emptyHandler{}

func setEnvVars() {
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "testFunction")
	_ = os.Setenv("AWS_REGION", "us-texas-1")
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	_ = os.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	_ = os.Setenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "128")
	_ = os.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
}

func TestLambdaHandlerSignatures(t *testing.T) {
	setEnvVars()

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
			lambdaHandler := InstrumentHandler(testCase.handler)
			handler := reflect.ValueOf(lambdaHandler)
			resp := handler.Call(testCase.args)
			assert.Len(t, resp, 2)
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
			lambdaHandler := InstrumentHandler(testCase.handler)
			handler := reflect.ValueOf(lambdaHandler)
			handlerType := handler.Type()

			var args []reflect.Value
			args = append(args, reflect.ValueOf(mockContext))
			if handlerType.NumIn() > 1 {
				args = append(args, reflect.ValueOf(testCase.input))
			}
			response := handler.Call(args)
			assert.Len(t, response, 2)
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
			handler := WrapHandler(lambda.NewHandler(testCase.handler))
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

func BenchmarkInstrumentHandler(b *testing.B) {
	setEnvVars()

	customerHandler := func(ctx context.Context, payload int) error {
		return nil
	}
	wrapped := InstrumentHandler(customerHandler)
	wrappedCallable := reflect.ValueOf(wrapped)
	ctx := reflect.ValueOf(mockContext)
	payload := reflect.ValueOf(0)
	args := []reflect.Value{ctx, payload}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrappedCallable.Call(args)
	}
}

func BenchmarkWrapHandler(b *testing.B) {
	setEnvVars()

	wrapped := WrapHandler(emptyHandler{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrapped.Invoke(mockContext, []byte{0})
	}
}
