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

package otellogrus

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
)

type Test struct {
	log     func(context.Context)
	require func(mocktrace.Event)
}

func TestLogrusHook(t *testing.T) {
	tests := []Test{
		{
			log: func(ctx context.Context) {
				logrus.WithContext(ctx).Info("hello")
			},
			require: func(event mocktrace.Event) {
				require.Equal(t, []label.KeyValue{
					logSeverityKey.String("INFO"),
					logMessageKey.String("hello"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				logrus.WithContext(ctx).WithField("foo", "bar").Warn("hello")
			},
			require: func(event mocktrace.Event) {
				require.Equal(t, []label.KeyValue{
					logSeverityKey.String("WARN"),
					logMessageKey.String("hello"),
					label.String("foo", "bar"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				err := errors.New("some error")
				logrus.WithContext(ctx).WithError(err).Error("hello")
			},
			require: func(event mocktrace.Event) {
				require.Equal(t, []label.KeyValue{
					logSeverityKey.String("ERROR"),
					logMessageKey.String("hello"),
					exceptionTypeKey.String("*errors.errorString"),
					exceptionMessageKey.String("some error"),
				}, event.Attributes)
			},
		},
		{
			log: func(ctx context.Context) {
				logrus.SetReportCaller(true)
				logrus.WithContext(ctx).Info("hello")
				logrus.SetReportCaller(false)
			},
			require: func(event mocktrace.Event) {
				set := label.NewSet(event.Attributes...)

				value, ok := set.Value(codeFunctionKey)
				require.True(t, ok)
				require.Contains(t, value.AsString(), "go.opentelemetry.io/contrib/extra/github.com/sirupsen/logrus/otellogrus")

				value, ok = set.Value(codeFilepathKey)
				require.True(t, ok)
				require.Contains(t, value.AsString(), "extra/github.com/sirupsen/logrus/otellogrus/logrus_test.go")

				_, ok = set.Value(codeLinenoKey)
				require.True(t, ok)
			},
		},
	}

	logrus.AddHook(NewLoggingHook(WithLevels(
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	)))
	tracer := mocktrace.NewTracer("test")

	for _, test := range tests {
		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "main")

		test.log(ctx)

		events := span.(*mocktrace.Span).Events
		require.Equal(t, 1, len(events))

		event := events[0]
		require.Equal(t, "log", event.Message)
		test.require(event)

		span.End()
	}
}

func TestSpanStatus(t *testing.T) {
	logrus.AddHook(NewLoggingHook())
	tracer := mocktrace.NewTracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "main")

	logrus.WithContext(ctx).Info("hello")
	require.Equal(t, codes.Unset, span.(*mocktrace.Span).Status)

	logrus.WithContext(ctx).Error("hello")
	require.Equal(t, codes.Error, span.(*mocktrace.Span).Status)
}
