// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

var (
	testBodyString = "log message"
	testSeverity   = log.SeverityInfo
	entry          = zapcore.Entry{
		Level:   zap.InfoLevel,
		Message: testBodyString,
	}
)

// embeddedLogger is a type alias so the embedded.Logger type doesn't conflict
// with the Logger method of the recorder when it is embedded.
type embeddedLogger = embedded.Logger // nolint:unused  // Used below.

// recorder records all [log.Record]s it is ased to emit.
type recorder struct {
	embedded.LoggerProvider
	embeddedLogger // nolint:unused  // Used to embed embedded.Logger.

	// Records are the records emitted.
	Record log.Record

	// Scope is the Logger scope recorder received when Logger was called.
	Scope instrumentation.Scope

	// MinSeverity is the minimum severity the recorder will return true for
	// when Enabled is called (unless enableKey is set).
	MinSeverity log.Severity
}

func (r *recorder) Logger(name string, opts ...log.LoggerOption) log.Logger {
	cfg := log.NewLoggerConfig(opts...)

	r.Scope = instrumentation.Scope{
		Name:      name,
		Version:   cfg.InstrumentationVersion(),
		SchemaURL: cfg.SchemaURL(),
	}
	return r
}

type enablerKey uint

var enableKey enablerKey

func (r *recorder) Enabled(ctx context.Context, record log.Record) bool {
	return ctx.Value(enableKey) != nil || record.Severity() >= r.MinSeverity
}

func (r *recorder) Emit(_ context.Context, record log.Record) {
	r.Record = record
}

// copied from field_test.go https://github.com/uber-go/zap/blob/b15585bc7a2b383592004f75df35fa2088db5481/zapcore/field_test.go#L39
// To create dummy object/array for zapcore.
type users int

func (u users) String() string {
	return fmt.Sprintf("%d users", int(u))
}

func (u users) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if int(u) < 0 {
		return errors.New("too few users")
	}
	enc.AddInt("users", int(u))
	return nil
}

func (u users) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if int(u) < 0 {
		return errors.New("too few users")
	}
	for i := 0; i < int(u); i++ {
		enc.AppendString("user")
	}
	return nil
}

// Copied from field_test.go. https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/memory_encoder_test.go
type turducken struct{}

func (t turducken) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddArray("ducks", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
		for i := 0; i < 2; i++ {
			err := arr.AppendObject(zapcore.ObjectMarshalerFunc(func(inner zapcore.ObjectEncoder) error {
				inner.AddString("in", "chicken")
				return nil
			}))
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

type turduckens int

func (t turduckens) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	tur := turducken{}
	for i := 0; i < int(t); i++ {
		err = multierr.Append(err, enc.AppendObject(tur))
	}
	return err
}

// maybeNamespace is an ObjectMarshaler that sometimes opens a namespace.
type maybeNamespace struct{ bool }

func (m maybeNamespace) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("obj-out", "obj-outside-namespace")
	if m.bool {
		enc.OpenNamespace("obj-namespace")
		enc.AddString("obj-in", "obj-inside-namespace")
	}
	return nil
}

type loggable struct{ bool }

func (l loggable) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AddString("loggable", "yes")
	return nil
}

func (l loggable) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AppendBool(true)
	return nil
}

// converts value to result.
func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		var s []any
		for _, val := range v.AsSlice() {
			s = append(s, value2Result(val))
		}
		return s
	case log.KindMap:
		m := make(map[string]any)
		for _, val := range v.AsMap() {
			m[val.Key] = value2Result(val.Value)
		}
		return m
	}
	return nil
}
