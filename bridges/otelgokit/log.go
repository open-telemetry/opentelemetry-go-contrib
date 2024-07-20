// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgokit // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"
	"fmt"
	"reflect"
	"time"

	gokitlog "github.com/go-kit/log"

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

// Option configures a [Core].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Core]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Core]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Core] to create its [log.Logger].
//
// By default if this Option is not provided, the Handler will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

type OTelLogger struct {
	logger log.Logger
	ctx    context.Context
}

var _ gokitlog.Logger = (*OTelLogger)(nil)

func NewOTelLogger(name string, opts ...Option) *OTelLogger {
	cfg := newConfig(opts)

	return &OTelLogger{
		logger: cfg.logger(name),
		ctx:    context.Background(),
	}
}

func (o *OTelLogger) Log(keyvals ...interface{}) error {
	r := log.Record{}
	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]

		if timeValue, ok := v.(time.Time); ok {
			r.SetTimestamp(timeValue)
			continue
		}

		if ctx, ok := v.(context.Context); ok {
			o.ctx = ctx
			continue
		}

		// TODO. Extract severity and severity text

		r.AddAttributes(log.KeyValue{Key: k.(string), Value: convertValue(v)})
	}
	o.logger.Emit(o.ctx, r)
	return nil
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
