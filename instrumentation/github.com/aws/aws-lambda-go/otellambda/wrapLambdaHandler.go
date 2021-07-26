package otellambda

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel"
)


func errorHandler(e error) func(context.Context, interface{}) (interface{}, error) {
	return func(context.Context, interface{}) (interface{}, error) {
		return nil, e
	}
}

// Ensure handler takes 0-2 values, with context
// as its first value if two arguments exist
func validateArguments(handler reflect.Type) (bool, error) {
	handlerTakesContext := false
	if handler.NumIn() > 2 {
		return false, fmt.Errorf("handlers may not take more than two arguments, but handler takes %d", handler.NumIn())
	} else if handler.NumIn() > 0 {
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		argumentType := handler.In(0)
		handlerTakesContext = argumentType.Implements(contextType)
		if handler.NumIn() > 1 && !handlerTakesContext {
			return false, fmt.Errorf("handler takes two arguments, but the first is not Context. got %s", argumentType.Kind())
		}
	}

	return handlerTakesContext, nil
}

// Ensure handler returns 0-2 values, with an error
// as its first value if any exist
func validateReturns(handler reflect.Type) error {
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	switch n := handler.NumOut(); {
	case n > 2:
		return fmt.Errorf("handler may not return more than two values")
	case n == 2:
		if !handler.Out(1).Implements(errorType) {
			return fmt.Errorf("handler returns two values, but the second does not implement error")
		}
	case n == 1:
		if !handler.Out(0).Implements(errorType) {
			return fmt.Errorf("handler returns a single value, but it does not implement error")
		}
	}

	return nil
}

// Wraps and calls customer lambda handler then unpacks response as necessary
func wrapperInternals(ctx context.Context, handlerFunc interface{}, event reflect.Value, takesContext bool) (interface{}, error) {
	wrappedLambdaHandler := reflect.ValueOf(wrapper(handlerFunc))

	argsWrapped := []reflect.Value{reflect.ValueOf(ctx), event, reflect.ValueOf(takesContext)}
	response := wrappedLambdaHandler.Call(argsWrapped)[0].Interface().([]reflect.Value)

	// convert return values into (interface{}, error)
	var err error
	if len(response) > 0 {
		if errVal, ok := response[len(response)-1].Interface().(error); ok {
			err = errVal
		}
	}
	var val interface{}
	if len(response) > 1 {
		val = response[0].Interface()
	}

	return val, err
}

// converts the given payload to the correct event type
func payloadToEvent(eventType reflect.Type, payload interface{}) (reflect.Value, error) {
	event := reflect.New(eventType)

	// lambda SDK normally unmarshalls to customer event type, however
	// with the wrapper the SDK unmarshalls to map[string]interface{}
	// due to our use of reflection. Therefore we must convert this map
	// to customer's desired event, we do so by simply re-marshalling then
	// unmarshalling to the desired event type
	remarshalledPayload, err := json.Marshal(payload)
	if err != nil {
		return reflect.Value{}, err
	}

	if err := json.Unmarshal(remarshalledPayload, event.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return event, nil
}

// WrapLambdaHandler Provides a lambda handler which wraps customer lambda handler with OTel Tracing
func WrapLambdaHandler(handlerFunc interface{}, options ...InstrumentationOption) interface{} {
	o := InstrumentationOptions{
		TracerProvider: otel.GetTracerProvider(),
		Flusher:        &noopFlusher{},
	}
	for _, opt := range options {
		opt(&o)
	}
	configuration = o

	if handlerFunc == nil {
		return errorHandler(fmt.Errorf("handler is nil"))
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
		return func(ctx context.Context) (interface{}, error) {
			var temp *interface{}
			event := reflect.ValueOf(temp)
			return wrapperInternals(ctx, handlerFunc, event, takesContext)
		}
	} else { // customer either takes both context and payload or just payload
		return func(ctx context.Context, payload interface{}) (interface{}, error) {
			event, err := payloadToEvent(handlerType.In(handlerType.NumIn()-1), payload)
			if err != nil {
				return nil, err
			}
			return wrapperInternals(ctx, handlerFunc, event.Elem(), takesContext)
		}
	}
}

// Adds OTel span surrounding customer handler call
func wrapper(handlerFunc interface{}) func(ctx context.Context, event interface{}, takesContext bool) []reflect.Value {
	return func(ctx context.Context, event interface{}, takesContext bool) []reflect.Value {

		ctx, span := tracingBegin(ctx)
		defer tracingEnd(ctx, span)

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

// Determine if an interface{} is nil or the
// if the reflect.Value of the event is nil
func eventExists(event interface{}) bool {
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

