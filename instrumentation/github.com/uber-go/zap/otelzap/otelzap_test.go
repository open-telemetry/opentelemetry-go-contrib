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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/oteltest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestWithContext(t *testing.T) {
	tests := []struct {
		log     func(context.Context)
		require func(oteltest.Event)
	}{
		{
			log: func(ctx context.Context) {
				InfoWithContext(ctx, "test_info", zap.String("test_info_key", "test_info_value"))
			},
			require: func(event oteltest.Event) {
				require.EqualValues(t, map[label.Key]label.Value{
					logMsg:          label.StringValue("test_info"),
					logLevel:        label.StringValue(zapcore.InfoLevel.CapitalString()),
					"test_info_key": label.StringValue("test_info_value"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				WarnWithContext(ctx, "test_warn", zap.String("test_warn_key", "test_warn_value"))
			},
			require: func(event oteltest.Event) {
				require.EqualValues(t, map[label.Key]label.Value{
					logMsg:          label.StringValue("test_warn"),
					logLevel:        label.StringValue(zapcore.WarnLevel.CapitalString()),
					"test_warn_key": label.StringValue("test_warn_value"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				ErrorWithContext(ctx, "test_error", zap.String("test_error_key", "test_error_value"))
			},
			require: func(event oteltest.Event) {
				require.EqualValues(t, map[label.Key]label.Value{
					logMsg:           label.StringValue("test_error"),
					logLevel:         label.StringValue(zapcore.ErrorLevel.CapitalString()),
					"test_error_key": label.StringValue("test_error_value"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				DPanicWithContext(ctx, "test_dpanic", zap.String("test_dpanic_key", "test_dpanic_value"))
			},
			require: func(event oteltest.Event) {
				require.EqualValues(t, map[label.Key]label.Value{
					logMsg:           label.StringValue("test_dpanic"),
					logLevel:         label.StringValue(zapcore.DPanicLevel.CapitalString()),
					"test_dpanic_key": label.StringValue("test_dpanic_value"),
				}, event.Attributes)
			},
		},
		// TODO: find a way around panic and fatal
		//{
		//	log: func(ctx context.Context) {
		//		PanicWithContext(ctx, "test_panic", zap.String("test_panic_key", "test_panic_value"))
		//	},
		//	require: func(event oteltest.Event) {
		//		require.EqualValues(t, map[label.Key]label.Value{
		//			logMsg:           label.StringValue("test_panic"),
		//			logLevel:         label.StringValue(zapcore.PanicLevel.CapitalString()),
		//			"test_panic_key": label.StringValue("test_panic_value"),
		//		}, event.Attributes)
		//	},
		//},
		//{
		//	log: func(ctx context.Context) {
		//		FatalWithContext(ctx, "test_fatal", zap.String("test_fatal_key", "test_fatal_value"))
		//	},
		//	require: func(event oteltest.Event) {
		//		require.EqualValues(t, map[label.Key]label.Value{
		//			logMsg:           label.StringValue("test_fatal"),
		//			logLevel:         label.StringValue(zapcore.PanicLevel.CapitalString()),
		//			"test_fatal_key": label.StringValue("test_fatal_value"),
		//		}, event.Attributes)
		//	},
		//},
	}

	log := zap.NewNop()
	zap.ReplaceGlobals(log)

	tp := oteltest.NewTracerProvider()
	tracer := tp.Tracer("test")

	for _, test := range tests {
		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "main")

		test.log(ctx)

		events := span.(*oteltest.Span).Events()
		require.Equal(t, 1, len(events))

		event := events[0]
		require.Equal(t, "log", event.Name)
		test.require(event)

		span.End()
	}
}
