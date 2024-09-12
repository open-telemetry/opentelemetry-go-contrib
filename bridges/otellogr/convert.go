package otellogr // import "go.opentelemetry.io/contrib/bridges/otellogr"

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/log"
)

// convertKVs converts a list of key-value pairs to a list of [log.KeyValue].
// The last [context.Context] value is returned as the context.
// If no context is found, the original context is returned.
func convertKVs(ctx context.Context, keysAndValues ...any) (context.Context, []log.KeyValue) {
	if len(keysAndValues) == 0 {
		return ctx, nil
	}
	if len(keysAndValues)%2 != 0 {
		// Ensure an odd number of items here does not corrupt the list
		keysAndValues = append(keysAndValues, nil)
	}

	kvs := make([]log.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		k, ok := keysAndValues[i].(string)
		if !ok {
			// Ensure that the key is a string
			k = fmt.Sprintf("%v", keysAndValues[i])
		}

		v := keysAndValues[i+1]
		if vCtx, ok := v.(context.Context); ok {
			// Special case when a field is of context.Context type.
			ctx = vCtx
			continue
		}

		kvs = append(kvs, log.KeyValue{
			Key:   k,
			Value: convertValue(v),
		})
	}

	return ctx, kvs
}

func convertValue(v any) log.Value {
	// Handling the most common types without reflect is a small perf win.
	switch val := v.(type) {
	case bool:
		return log.BoolValue(val)
	case string:
		return log.StringValue(val)
	case int:
		return log.Int64Value(int64(val))
	case int8:
		return log.Int64Value(int64(val))
	case int16:
		return log.Int64Value(int64(val))
	case int32:
		return log.Int64Value(int64(val))
	case int64:
		return log.Int64Value(val)
	case uint:
		return convertUintValue(uint64(val))
	case uint8:
		return log.Int64Value(int64(val))
	case uint16:
		return log.Int64Value(int64(val))
	case uint32:
		return log.Int64Value(int64(val))
	case uint64:
		return convertUintValue(val)
	case uintptr:
		return convertUintValue(uint64(val))
	case float32:
		return log.Float64Value(float64(val))
	case float64:
		return log.Float64Value(val)
	case time.Duration:
		return log.Int64Value(val.Nanoseconds())
	case complex64:
		r := log.Float64("r", real(complex128(val)))
		i := log.Float64("i", imag(complex128(val)))
		return log.MapValue(r, i)
	case complex128:
		r := log.Float64("r", real(val))
		i := log.Float64("i", imag(val))
		return log.MapValue(r, i)
	case time.Time:
		return log.Int64Value(val.UnixNano())
	case []byte:
		return log.BytesValue(val)
	case error:
		return log.StringValue(val.Error())
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return log.Value{}
	}
	val := reflect.ValueOf(v)
	switch t.Kind() {
	case reflect.Struct:
		return log.StringValue(fmt.Sprintf("%+v", v))
	case reflect.Slice, reflect.Array:
		items := make([]log.Value, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			items = append(items, convertValue(val.Index(i).Interface()))
		}
		return log.SliceValue(items...)
	case reflect.Map:
		kvs := make([]log.KeyValue, 0, val.Len())
		for _, k := range val.MapKeys() {
			var key string
			// If the key is a struct, use %+v to print the struct fields.
			if k.Kind() == reflect.Struct {
				key = fmt.Sprintf("%+v", k.Interface())
			} else {
				key = fmt.Sprintf("%v", k.Interface())
			}
			kvs = append(kvs, log.KeyValue{
				Key:   key,
				Value: convertValue(val.MapIndex(k).Interface()),
			})
		}
		return log.MapValue(kvs...)
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return log.Value{}
		}
		return convertValue(val.Elem().Interface())
	}

	// Try to handle this as gracefully as possible.
	//
	// Don't panic here. it is preferable to have user's open issue
	// asking why their attributes have a "unhandled: " prefix than
	// say that their code is panicking.
	return log.StringValue(fmt.Sprintf("unhandled: (%s) %+v", t, v))
}

// convertUintValue converts a uint64 to a log.Value.
// If the value is too large to fit in an int64, it is converted to a string.
func convertUintValue(v uint64) log.Value {
	if v > math.MaxInt64 {
		return log.StringValue(strconv.FormatUint(v, 10))
	}
	return log.Int64Value(int64(v))
}
