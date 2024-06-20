// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzerolog provides a SeverityHook, a zerolog.Hook implementation that
// can be used to bridge between the [zerolog] API and [OpenTelemetry].
//
// # Record Conversion
//
// The zerolog.Event records are converted to OpenTelemetry [log.Record] in
// the following way:
//
//   - Time is set as the Timestamp.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - Fields are transformed and set as the attributes.
//
// The Level is transformed by using a mapping function to the OpenTelemetry
// Severity types. For example:
//
//   - zerolog.DebugLevel is transformed to [log.SeverityDebug]
//   - zerolog.InfoLevel is transformed to [log.SeverityInfo]
//   - zerolog.WarnLevel is transformed to [log.SeverityWarn]
//   - zerolog.ErrorLevel is transformed to [log.SeverityError]
//   - zerolog.FatalLevel and zerolog.PanicLevel are mapped to
//     [log.SeverityError] (consider customization for these levels)
//
// Attribute values are transformed based on their type into log attributes, or
// into a string value if there is no matching type.
//
// [zerolog]: https://github.com/rs/zerolog
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otelzerolog // import "go.opentelemetry.io/contrib/bridges/otelzerolog"

import (
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	return c.provider.Logger(name, opts...)
}

// Option configures a SeverityHook.
type Option interface {
	apply(config) config
}
type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [SeverityHook]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [SeverityHook]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [SeverityHook].
//
// By default if this Option is not provided, the SeverityHook will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// NewSeverityHook returns a new [SeverityHook] to be used as a [Zerolog.Hook].
//
// If [WithLoggerProvider] is not provided, the returned SeverityHook will use the
// global LoggerProvider.
func NewSeverityHook(name string, options ...Option) *SeverityHook {
	cfg := newConfig(options)
	return &SeverityHook{
		logger: cfg.logger(name),
	}
}

// SeverityHook is a [zerolog.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type SeverityHook struct {
	logger log.Logger
	level  zerolog.Level
}

// Levels returns the list of log levels we want to be sent to OpenTelemetry.
func (h *SeverityHook) Level() zerolog.Level {
	return h.level
}

// Run handles the passed record, and sends it to OpenTelemetry.
func (h SeverityHook) Run(e *zerolog.Event, level zerolog.Level, msg string) error {
	if level != zerolog.NoLevel {
		e.Str(level.String(), msg)
	}
	h.logger.Emit(e.GetCtx(), h.convertEvent(e, level, msg))
	return nil
}

func (h *SeverityHook) convertEvent(e *zerolog.Event, level zerolog.Level, msg string) log.Record {
	var record log.Record
	record.SetTimestamp(zerolog.TimestampFunc())
	record.SetBody(log.StringValue(msg))
	record.SetSeverity(log.Severity(level))
	record.SetObservedTimestamp(time.Now())
	record.AddAttributes(convertFields(e)...)
	return record
}

func convertFields(e *zerolog.Event) []log.KeyValue {
	kvs := make([]log.KeyValue, 0)

	// Extract fields from the event and convert them
	e.Fields(func(key string, value interface{}) {
		kvs = append(kvs, log.KeyValue{
			Key:   key,
			Value: convertValue(value),
		})
	})

	return kvs
}

func convertValue(v interface{}) log.Value {
	switch v := v.(type) {
	case bool:
		return log.BoolValue(v)
	case []byte:
		return log.BytesValue(v)
	case float64:
		return log.Float64Value(v)
	case int:
		return log.IntValue(v)
	case int64:
		return log.Int64Value(v)
	case string:
		return log.StringValue(v)
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
		return convertValue(val.Elem().Interface())
	}

	return log.StringValue(fmt.Sprintf("unhandled attribute type: (%s) %+v", t, v))
}