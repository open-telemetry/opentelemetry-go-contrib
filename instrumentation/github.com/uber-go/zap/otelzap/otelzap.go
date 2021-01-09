// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Based on https://github.com/opentracing-contrib/go-zap
package otelzap

import (
	"context"

	otellabel "go.opentelemetry.io/otel/label"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DebugWithContext logs on debug level and trace based on the context span if it exists.
func DebugWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	DebugWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// DebugWithSpan logs on debug level and add the logs on the trace if span exists.
func DebugWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	Debug(log, fields...)
	logSpan(span, log, fields...)
}

// Debug logs on debug level.
func Debug(log string, fields ...zapcore.Field) {
	zap.L().Debug(log, fields...)
}

// InfoWithContext logs on info level and trace based on the context span if it exists.
func InfoWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	InfoWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// InfoWithSpan logs on info level and add the logs on the trace if span exists.
func InfoWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	Info(log, fields...)
	logSpan(span, log, fields...)

}

// Info logs on info level.
func Info(log string, fields ...zapcore.Field) {
	zap.L().Info(log, fields...)
}

// WarnWithContext logs on warn level and trace based on the context span if it exists.
func WarnWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	WarnWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// WarnWithSpan logs on warn level and add the logs on the trace if span exists.
func WarnWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	Warn(log, fields...)
	logSpan(span, log, fields...)

}

// Warn logs on warn level.
func Warn(log string, fields ...zapcore.Field) {
	zap.L().Warn(log, fields...)
}

// ErrorWithContext logs on error level and trace based on the context span if it exists.
func ErrorWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	ErrorWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// ErrorWithSpan logs on error level and add the logs on the trace if span exists.
func ErrorWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	Error(log, fields...)
	logSpan(span, log, fields...)
}

// Error logs on error level.
func Error(log string, fields ...zapcore.Field) {
	zap.L().Error(log, fields...)
}


// DPanicWithContext logs on dPanic level and trace based on the context span if it exists.
func DPanicWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	DPanicWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// DPanicWithSpan logs on dPanic level and add the logs on the trace if span exists.
func DPanicWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	logSpan(span, log, fields...)
	DPanic(log, fields...)
}

// DPanic logs on dPanic level.
func DPanic(log string, fields ...zapcore.Field) {
	zap.L().DPanic(log, fields...)
}

// PanicWithContext logs on panic level and trace based on the context span if it exists.
func PanicWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	PanicWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// PanicWithSpan logs on panic level and add the logs on the trace if span exists.
func PanicWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	logSpan(span, log, fields...)
	Panic(log, fields...)
}

// Panic logs on panic level.
func Panic(log string, fields ...zapcore.Field) {
	zap.L().Panic(log, fields...)
}

// FatalWithContext logs on fatal level and trace based on the context span if it exists.
func FatalWithContext(ctx context.Context, log string, fields ...zapcore.Field) {
	FatalWithSpan(oteltrace.SpanFromContext(ctx), log, fields...)
}

// FatalWithSpan logs on fatal level and add the logs on the trace if span exists.
func FatalWithSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	logSpan(span, log, fields...)
	Fatal(log, fields...)
}

// Fatal logs on fatal level.
func Fatal(log string, fields ...zapcore.Field) {
	zap.L().Fatal(log, fields...)
}

func logSpan(span oteltrace.Span, log string, fields ...zapcore.Field) {
	if span != nil {
		attrs := make([]otellabel.KeyValue, len(fields)+1)
		if log != "" {
			attrs = append(attrs, otellabel.String("event", log))
		}
		if len(fields) > 0 {
			attrs = append(attrs, zapFieldsToOtel(fields...)...)
		}
		span.SetAttributes(attrs...)
	}
}