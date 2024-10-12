// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogrus provides a [Hook], a [logrus.Hook] implementation that
// can be used to bridge between the [github.com/sirupsen/logrus] API and
// [OpenTelemetry].
//
// # Record Conversion
//
// The [logrus.Entry] records are converted to OpenTelemetry [log.Record] in
// the following way:
//
//   - Time is set as the Timestamp.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - Fields are transformed and set as the attributes.
//
// The Level is transformed to the OpenTelemetry
// Severity types. For example:
//
//   - [logrus.DebugLevel] is transformed to [log.SeverityDebug]
//   - [logrus.InfoLevel] is transformed to [log.SeverityInfo]
//   - [logrus.WarnLevel] is transformed to [log.SeverityWarn]
//   - [logrus.ErrorLevel] is transformed to [log.SeverityError]
//   - [logrus.FatalLevel] is transformed to [log.SeverityFatal]
//   - [logrus.PanicLevel] is transformed to [log.SeverityFatal4]
//
// Field values are transformed based on their type into log attributes, or
// into a string value if there is no matching type.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otellogrus // import "go.opentelemetry.io/contrib/bridges/otellogrus"

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string

	levels []logrus.Level
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	if c.levels == nil {
		c.levels = logrus.AllLevels
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

// Option configures a [Hook].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Hook]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Hook]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Hook].
//
// By default if this Option is not provided, the Hook will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithLevels returns an [Option] that configures the log levels that will fire
// the configured [Hook].
//
// By default if this Option is not provided, the Hook will fire for all levels.
// LoggerProvider.
func WithLevels(l []logrus.Level) Option {
	return optFunc(func(c config) config {
		c.levels = l
		return c
	})
}

// NewHook returns a new [Hook] to be used as a [logrus.Hook].
//
// If [WithLoggerProvider] is not provided, the returned Hook will use the
// global LoggerProvider.
func NewHook(name string, options ...Option) *Hook {
	cfg := newConfig(options)
	return &Hook{
		logger: cfg.logger(name),
		levels: cfg.levels,
	}
}

// Hook is a [logrus.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type Hook struct {
	logger log.Logger
	levels []logrus.Level
}

// Levels returns the list of log levels we want to be sent to OpenTelemetry.
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// Fire handles the passed record, and sends it to OpenTelemetry.
func (h *Hook) Fire(entry *logrus.Entry) error {
	ctx := entry.Context
	h.logger.Emit(ctx, h.convertEntry(entry))
	return nil
}

func (h *Hook) convertEntry(e *logrus.Entry) log.Record {
	var record log.Record
	record.SetTimestamp(e.Time)
	record.SetBody(log.StringValue(e.Message))
	record.SetSeverity(convertSeverity(e.Level))
	record.AddAttributes(convertFields(e.Data)...)

	return record
}

func convertFields(fields logrus.Fields) []log.KeyValue {
	kvs := make([]log.KeyValue, 0, len(fields))
	for k, v := range fields {
		kvs = append(kvs, log.KeyValue{
			Key:   k,
			Value: convertValue(v),
		})
	}
	return kvs
}

func convertSeverity(level logrus.Level) log.Severity {
	switch level {
	case logrus.PanicLevel:
		// PanicLevel is not supported by OpenTelemetry, use Fatal4 as the highest severity.
		return log.SeverityFatal4
	case logrus.FatalLevel:
		return log.SeverityFatal
	case logrus.ErrorLevel:
		return log.SeverityError
	case logrus.WarnLevel:
		return log.SeverityWarn
	case logrus.InfoLevel:
		return log.SeverityInfo
	case logrus.DebugLevel:
		return log.SeverityDebug
	case logrus.TraceLevel:
		return log.SeverityTrace
	default:
		// If the level is not recognized, use SeverityUndefined as the lowest severity.
		// we should never reach this point as logrus only uses the above levels.
		return log.SeverityUndefined
	}
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
		if val.IsNil() {
			return log.Value{}
		}
		return convertValue(val.Elem().Interface())
	}

	return log.StringValue(fmt.Sprintf("unhandled attribute type: (%s) %+v", t, v))
}
