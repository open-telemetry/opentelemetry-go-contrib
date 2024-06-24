// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogr provides [LogSink], an [logr.LogSink] implementation, that
// can be used to bridge between the [github.com/go-logr/logr] API and
// [OpenTelemetry].
//
// # Record Conversion
//
// The logr records are converted to OpenTelemetry [log.Record] in the following
// way:
//
//   - Time is set as the current time of conversion.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - PC is dropped.
//   - KeyAndValues are transformed and set as Attributes.
//   - Error is always logged as an additional attribute with the key "err" and
//     with the severity [log.SeverityError].
//   - Name is logged as an additional attribute with the key "logger".
//   - The [context.Context] value in KeyAndValues is propagated to OpenTelemetry
//     log record. All non-nested [context.Context] values are ignored and not
//     added as attributes. If there are multiple [context.Context] the last one
//     is used.
//
// The Level is transformed by using the [WithLevelSeverity] option. If this is
// not provided it would default to a function that adds an offset to the logr
// such that [logr.Info] is transformed to [log.SeverityInfo]. For example:
//
//   - [logr.Info] is transformed to [log.SeverityInfo].
//   - [logr.V(0)] is transformed to [log.SeverityInfo].
//   - [logr.V(1)] is transformed to [log.SeverityInfo2].
//   - [logr.V(2)] is transformed to [log.SeverityInfo3].
//   - ...
//   - [logr.V(15)] is transformed to [log.SeverityFatal4].
//   - [logr.Error] is transformed to [log.SeverityError].
//
// KeysAndValues values are transformed based on their type. The following types are
// supported:
//
//   - [bool] are transformed to [log.BoolValue].
//   - [string] are transformed to [log.StringValue].
//   - [int], [int8], [int16], [int32], [int64] are transformed to
//     [log.Int64Value].
//   - [uint], [uint8], [uint16], [uint32], [uint64], [uintptr] are transformed
//     to [log.Int64Value] or [log.StringValue] if the value is too large.
//   - [float32], [float64] are transformed to [log.Float64Value].
//   - [time.Duration] are transformed to [log.Int64Value] with the nanoseconds.
//   - [complex64], [complex128] are transformed to [log.StringValue] with the
//   - [time.Time] are transformed to [log.Int64Value] with the nanoseconds.
//   - [[]byte] are transformed to [log.BytesValue].
//   - [error] are transformed to [log.StringValue] with the error message.
//   - [nil] are transformed to an empty [log.Value].
//   - [struct] are transformed to [log.StringValue] with the struct fields.
//   - [slice], [array] are transformed to [log.SliceValue] with the elements.
//   - [map] are transformed to [log.MapValue] with the key-value pairs.
//   - [pointer], [interface] are transformed to the dereferenced value.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otellogr // import "go.opentelemetry.io/contrib/bridges/otellogr"

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridges/otellogr"
	// errorKey is used to log the error parameter of Error as an additional attribute.
	errorKey = "error"
)

type config struct {
	provider      log.LoggerProvider
	scope         instrumentation.Scope
	levelSeverity func(int) log.Severity
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	var emptyScope instrumentation.Scope
	if c.scope == emptyScope {
		c.scope = instrumentation.Scope{
			Name:    bridgeName,
			Version: version,
		}
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	if c.levelSeverity == nil {
		c.levelSeverity = func(level int) log.Severity {
			const sevOffset = int(log.SeverityInfo)
			return log.Severity(level + sevOffset)
		}
	}

	return c
}

func (c config) logger() log.Logger {
	var opts []log.LoggerOption
	if c.scope.Version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.scope.Version))
	}
	if c.scope.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.scope.SchemaURL))
	}
	return c.provider.Logger(c.scope.Name, opts...)
}

// Option configures a [LogSink].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an [Option] that configures the scope of
// the [log.Logger] used by a [LogSink].
//
// By default if this Option is not provided, the LogSink will use a default
// instrumentation scope describing this bridge package. It is recommended to
// provide this so log data can be associated with its source package or
// module.
func WithInstrumentationScope(scope instrumentation.Scope) Option {
	return optFunc(func(c config) config {
		c.scope = scope
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [LogSink] to create its [log.Logger].
//
// By default if this Option is not provided, the LogSink will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithLevelSeverity returns an [Option] that configures the function used to
// convert logr levels to OpenTelemetry log severities.
//
// By default if this Option is not provided, the LogSink will use a default
// conversion function which adds an offset to the logr level to get the
// OpenTelemetry severity. The offset is such that logr.Info("message")
// converts to OpenTelemetry [log.SeverityInfo].
func WithLevelSeverity(f func(int) log.Severity) Option {
	return optFunc(func(c config) config {
		c.levelSeverity = f
		return c
	})
}

// NewLogSink returns a new [LogSink] to be used as a [logr.LogSink].
//
// If [WithLoggerProvider] is not provided, the returned LogSink will use the
// global LoggerProvider.
func NewLogSink(options ...Option) *LogSink {
	c := newConfig(options)
	return &LogSink{
		config:        c,
		logger:        c.logger(),
		levelSeverity: c.levelSeverity,
	}
}

// LogSink is a [logr.LogSink] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type LogSink struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	config        config
	logger        log.Logger
	levelSeverity func(int) log.Severity
	values        []log.KeyValue
}

// Compile-time check *Handler implements logr.LogSink.
var _ logr.LogSink = (*LogSink)(nil)

// log sends a log record to the OpenTelemetry logger.
func (l *LogSink) log(err error, msg string, serverity log.Severity, kvList ...any) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetBody(log.StringValue(msg))
	record.SetSeverity(serverity)

	if err != nil {
		record.AddAttributes(log.KeyValue{
			Key:   errorKey,
			Value: convertValue(err),
		})
	}

	if len(l.values) > 0 {
		record.AddAttributes(l.values...)
	}

	ctx, kv := convertKVs(kvList)
	if len(kv) > 0 {
		record.AddAttributes(kv...)
	}

	l.logger.Emit(ctx, record)
}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (l *LogSink) Enabled(level int) bool {
	var record log.Record
	record.SetSeverity(l.levelSeverity(level))
	ctx := context.Background()
	return l.logger.Enabled(ctx, record)
}

// Error logs an error, with the given message and key/value pairs.
func (l *LogSink) Error(err error, msg string, keysAndValues ...any) {
	l.log(err, msg, log.SeverityError, keysAndValues...)
}

// Info logs a non-error message with the given key/value pairs.
func (l *LogSink) Info(level int, msg string, keysAndValues ...any) {
	l.log(nil, msg, l.levelSeverity(level), keysAndValues...)
}

// Init receives optional information about the logr library this
// implementation does not use it.
func (l *LogSink) Init(info logr.RuntimeInfo) {
	// We don't need to do anything here.
	// CallDepth is used to calculate the caller's PC.
	// PC is dropped as part of the conversion to the OpenTelemetry log.Record.
}

// WithName returns a new LogSink with the specified name appended.
func (l LogSink) WithName(name string) logr.LogSink {
	newConfig := l.config
	newConfig.scope.Name = fmt.Sprintf("%s/%s", l.config.scope.Name, name)

	return &LogSink{
		config:        newConfig,
		logger:        newConfig.logger(),
		levelSeverity: newConfig.levelSeverity,
		values:        l.values,
	}
}

// WithValues returns a new LogSink with additional key/value pairs.
func (l LogSink) WithValues(keysAndValues ...any) logr.LogSink {
	_, attrs := convertKVs(keysAndValues)
	l.values = append(l.values, attrs...)
	return &l
}

// convertKVs converts a list of key-value pairs to a list of [log.KeyValue].
// The last [context.Context] value is returned as the context.
func convertKVs(keysAndValues []any) (context.Context, []log.KeyValue) {
	ctx := context.Background()

	if len(keysAndValues) == 0 {
		return ctx, nil
	}
	if len(keysAndValues)%2 != 0 {
		// Ensure an odd number of items here does not corrupt the list
		keysAndValues = append(keysAndValues, nil)
	}

	kv := make([]log.KeyValue, 0, len(keysAndValues)/2)
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

		kv = append(kv, log.KeyValue{
			Key:   k,
			Value: convertValue(v),
		})
	}
	return ctx, kv
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
		return log.StringValue(strconv.FormatComplex(complex128(val), 'f', -1, 64))
	case complex128:
		return log.StringValue(strconv.FormatComplex(val, 'f', -1, 128))
	case time.Time:
		return log.Int64Value(val.UnixNano())
	case []byte:
		return log.BytesValue(val)
	case error:
		return log.StringValue(fmt.Sprintf("%+v", val))
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
