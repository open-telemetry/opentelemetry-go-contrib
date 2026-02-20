// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// wrappedHandlerFunction is a struct which only holds an instrumentor and is
// able to instrument invocations of the user's lambda handler function.
type wrappedHandlerFunction struct {
	instrumentor instrumentor
}

func errorHandler(e error) func(context.Context, any) (any, error) {
	return func(context.Context, any) (any, error) {
		return nil, e
	}
}

// Ensure handler takes 0-2 values, with context
// as its first value if two arguments exist.
func validateArguments(handler reflect.Type) (bool, error) {
	handlerTakesContext := false
	if handler.NumIn() > 2 {
		return false, fmt.Errorf("handlers may not take more than two arguments, but handler takes %d", handler.NumIn())
	} else if handler.NumIn() > 0 {
		contextType := reflect.TypeFor[context.Context]()
		argumentType := handler.In(0)
		handlerTakesContext = argumentType.Implements(contextType)
		if handler.NumIn() > 1 && !handlerTakesContext {
			return false, fmt.Errorf("handler takes two arguments, but the first is not Context. got %s", argumentType.Kind())
		}
	}

	return handlerTakesContext, nil
}

// Ensure handler returns 0-2 values, with an error
// as its first value if any exist.
func validateReturns(handler reflect.Type) error {
	errorType := reflect.TypeFor[error]()

	switch n := handler.NumOut(); {
	case n > 2:
		return errors.New("handler may not return more than two values")
	case n == 2:
		if !handler.Out(1).Implements(errorType) {
			return errors.New("handler returns two values, but the second does not implement error")
		}
	case n == 1:
		if !handler.Out(0).Implements(errorType) {
			return errors.New("handler returns a single value, but it does not implement error")
		}
	}

	return nil
}

// Wraps and calls customer lambda handler then unpacks response as necessary.
func (whf *wrappedHandlerFunction) wrapperInternals(ctx context.Context, handlerFunc any, eventJSON []byte, event reflect.Value, takesContext bool) (any, error) {
	wrappedLambdaHandler := reflect.ValueOf(whf.wrapper(handlerFunc))

	argsWrapped := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(eventJSON), event, reflect.ValueOf(takesContext)}
	response := wrappedLambdaHandler.Call(argsWrapped)[0].Interface().([]reflect.Value)

	// convert return values into (any, error)
	var err error
	if len(response) > 0 {
		if errVal, ok := response[len(response)-1].Interface().(error); ok {
			err = errVal
		}
	}
	var val any
	if len(response) > 1 {
		val = response[0].Interface()
	}

	return val, err
}

// InstrumentHandler Provides a lambda handler which wraps customer lambda handler with OTel Tracing.
func InstrumentHandler(handlerFunc any, options ...Option) any {
	whf := wrappedHandlerFunction{instrumentor: newInstrumentor(options...)}

	if handlerFunc == nil {
		return errorHandler(errors.New("handler is nil"))
	}
	handlerType := reflect.TypeOf(handlerFunc)
	if handlerType.Kind() != reflect.Func {
		return errorHandler(fmt.Errorf("handler kind %s is not %s", handlerType.Kind(), reflect.Func))
	}

	takesContext, err := validateArguments(handlerType)
	if err != nil {
		return errorHandler(err)
	}

	if err := validateReturns(handlerType); err != nil {
		return errorHandler(err)
	}

	// note we will always take context to capture lambda context,
	// regardless of whether customer takes context
	if handlerType.NumIn() == 0 || handlerType.NumIn() == 1 && takesContext {
		return func(ctx context.Context) (any, error) {
			var temp *any
			event := reflect.ValueOf(temp)
			return whf.wrapperInternals(ctx, handlerFunc, []byte{}, event, takesContext)
		}
	}

	// customer either takes both context and payload or just payload
	return func(ctx context.Context, payload any) (any, error) {
		event := reflect.New(handlerType.In(handlerType.NumIn() - 1))

		// lambda SDK normally unmarshalls to customer event type, however
		// with the wrapper the SDK unmarshalls to map[string]any
		// due to our use of reflection. Therefore we must convert this map
		// to customer's desired event, we do so by simply re-marshaling then
		// unmarshalling to the desired event type. The remarshalledPayload
		// will also be used by users using custom propagators
		remarshalledPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(remarshalledPayload, event.Interface()); err != nil {
			return nil, err
		}

		return whf.wrapperInternals(ctx, handlerFunc, remarshalledPayload, event.Elem(), takesContext)
	}
}

// Adds OTel span surrounding customer handler call.
func (whf *wrappedHandlerFunction) wrapper(handlerFunc any) func(ctx context.Context, eventJSON []byte, event any, takesContext bool) []reflect.Value {
	return func(ctx context.Context, eventJSON []byte, event any, takesContext bool) []reflect.Value {
		ctx, span := whf.instrumentor.tracingBegin(ctx, eventJSON)
		defer whf.instrumentor.tracingEnd(ctx, span)

		handler := reflect.ValueOf(handlerFunc)
		var args []reflect.Value
		if takesContext {
			args = append(args, reflect.ValueOf(ctx))
		}
		if eventExists(event) {
			args = append(args, reflect.ValueOf(event))
		}

		response := handler.Call(args)

		return response
	}
}

// Determine if an any is nil or the
// if the reflect.Value of the event is nil.
func eventExists(event any) bool {
	if event == nil {
		return false
	}

	// reflect.Value.isNil() can only be called on
	// Values of certain Kinds. Unsupported Kinds
	// will panic rather than return false
	switch reflect.TypeOf(event).Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return !reflect.ValueOf(event).IsNil()
	}
	return true
}
