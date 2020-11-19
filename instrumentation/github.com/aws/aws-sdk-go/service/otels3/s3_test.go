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

package otels3

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/otels3/mocks"
	mockmetric "go.opentelemetry.io/contrib/internal/metric"
	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

var (
	s3bucket = "s3bucket"
)

func getLabelValFromMeasurementBatch(key label.Key, batch mockmetric.Batch) *label.Value {
	for _, label := range batch.Labels {
		if label.Key == key {
			return &label.Value
		}
	}
	return nil
}

func getLabelValFromSpan(key label.Key, span mocktrace.Span) *label.Value {
	if value, ok := span.Attributes[key]; ok {
		return &value
	}
	return nil
}

func assertMetrics(t *testing.T, mockedMeterImp *mockmetric.MeterImpl) {
	// In Meter we have one duration recorder, one operation counter
	assert.Equal(t, 2, len(mockedMeterImp.MeasurementBatches))

	metricsFound := map[string]bool{
		"aws.s3.operation.duration": false,
		"aws.s3.operation":          false,
	}
	// iterate over metrics to get names
	for _, measurementBatch := range mockedMeterImp.MeasurementBatches {
		for _, measurement := range measurementBatch.Measurements {
			metricName := measurement.Instrument.Descriptor().Name()
			//check if we are looking for this metric name, if so, mark as found
			if _, ok := metricsFound[metricName]; ok {
				metricsFound[metricName] = true
			}
		}
	}

	//check all metric names are found
	for metricName, metricFound := range metricsFound {
		assert.True(t, metricFound, fmt.Sprintf("should find metric %s", metricName))
	}
}

func assertSpanCorrelation(t *testing.T, spanCorrelation bool, mockedMeterImp *mockmetric.MeterImpl, span *mocktrace.Span) {
	for _, measurementBatch := range mockedMeterImp.MeasurementBatches {
		if spanCorrelation {
			traceID := span.SpanContext().TraceID.String()
			spanID := span.SpanContext().SpanID.String()

			assert.Equal(t, traceID, getLabelValFromMeasurementBatch("trace.id", measurementBatch).AsString())
			assert.Equal(t, spanID, getLabelValFromMeasurementBatch("span.id", measurementBatch).AsString())
		} else {
			assert.Nil(t, getLabelValFromMeasurementBatch("trace.id", measurementBatch))
			assert.Nil(t, getLabelValFromMeasurementBatch("span.id", measurementBatch))
		}
	}
}

func Test_instrumentedS3_PutObjectWithContext(t *testing.T) {
	type fields struct {
		spanCorrelation bool
		mockSetup       func() (expectedReturn interface{})
	}
	type args struct {
		ctx   aws.Context
		input *s3.PutObjectInput
		opts  []request.Option
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "instrumentedS3.PutObjectWithContext should be delegated to S3.PutObjectWithContext while metrics and spans are linked",
			fields: fields{
				spanCorrelation: true,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.PutObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.PutObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
		{
			name: "instrumentedS3.PutObjectWithContext should be delegated to S3.PutObjectWithContext while metrics and spans are NOT linked",
			fields: fields{
				spanCorrelation: false,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.PutObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.PutObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, mockedTracer := mocktrace.NewTracerProviderAndTracer(instrumentationName)
			mockedMeterImp, mockedMeter := mockmetric.NewMeter()
			mockedCounters := createCounters(mockedMeter)
			mockedRecorders := createRecorders(mockedMeter)
			mockedPropagators := global.TextMapPropagator()

			s3Mock := &mocks.MockS3Client{}
			s := &instrumentedS3{
				S3API:           s3Mock,
				tracer:          mockedTracer,
				meter:           mockedMeter,
				propagators:     mockedPropagators,
				counters:        mockedCounters,
				recorders:       mockedRecorders,
				spanCorrelation: tt.fields.spanCorrelation,
			}
			expectedReturn := tt.fields.mockSetup()
			got, err := s.PutObjectWithContext(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutObjectWithContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, expectedReturn) {
				t.Errorf("PutObjectWithContext() got = %v, want %v", got, expectedReturn)
			}
			spans := mockedTracer.EndedSpans()
			assert.Equal(t, 1, len(spans))
			assert.Equal(t, trace.SpanKindClient, spans[0].Kind)
			assert.Equal(t, s3StorageSystemValue, getLabelValFromSpan(storageSystemKey, *spans[0]).AsString())
			assert.Equal(t, *tt.args.input.Bucket, getLabelValFromSpan(storageDestinationKey, *spans[0]).AsString())
			assert.Equal(t, operationPutObject, getLabelValFromSpan(storageOperationKey, *spans[0]).AsString())

			// In Meter we have one duration recorder, one operation counter
			assert.Equal(t, 2, len(mockedMeterImp.MeasurementBatches))

			assertMetrics(t, mockedMeterImp)

			assertSpanCorrelation(t, tt.fields.spanCorrelation, mockedMeterImp, spans[0])
		})
	}
}

func Test_instrumentedS3_GetObjectWithContext(t *testing.T) {
	type fields struct {
		spanCorrelation bool
		mockSetup       func() (expectedReturn interface{})
	}
	type args struct {
		ctx   aws.Context
		input *s3.GetObjectInput
		opts  []request.Option
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "instrumentedS3.GetObjectWithContext should be delegated to S3.GetObjectWithContext while metrics and spans are linked",
			fields: fields{
				spanCorrelation: true,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.GetObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.GetObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
		{
			name: "instrumentedS3.GetObjectWithContext should be delegated to S3.GetObjectWithContext while metrics and spans are NOT linked",
			fields: fields{
				spanCorrelation: false,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.GetObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.GetObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, mockedTracer := mocktrace.NewTracerProviderAndTracer(instrumentationName)
			mockedMeterImp, mockedMeter := mockmetric.NewMeter()
			mockedCounters := createCounters(mockedMeter)
			mockedRecorders := createRecorders(mockedMeter)
			mockedPropagators := global.TextMapPropagator()

			s3Mock := &mocks.MockS3Client{}
			s := &instrumentedS3{
				S3API:           s3Mock,
				tracer:          mockedTracer,
				meter:           mockedMeter,
				propagators:     mockedPropagators,
				counters:        mockedCounters,
				recorders:       mockedRecorders,
				spanCorrelation: tt.fields.spanCorrelation,
			}
			expectedReturn := tt.fields.mockSetup()
			got, err := s.GetObjectWithContext(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetObjectWithContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, expectedReturn) {
				t.Errorf("GetObjectWithContext() got = %v, want %v", got, expectedReturn)
			}
			spans := mockedTracer.EndedSpans()
			assert.Equal(t, 1, len(spans))
			assert.Equal(t, trace.SpanKindClient, spans[0].Kind)
			assert.Equal(t, s3StorageSystemValue, getLabelValFromSpan(storageSystemKey, *spans[0]).AsString())
			assert.Equal(t, *tt.args.input.Bucket, getLabelValFromSpan(storageDestinationKey, *spans[0]).AsString())
			assert.Equal(t, operationGetObject, getLabelValFromSpan(storageOperationKey, *spans[0]).AsString())

			// In Meter we have one duration recorder, one operation counter
			assert.Equal(t, 2, len(mockedMeterImp.MeasurementBatches))

			assertMetrics(t, mockedMeterImp)

			assertSpanCorrelation(t, tt.fields.spanCorrelation, mockedMeterImp, spans[0])
		})
	}
}

func Test_instrumentedS3_DeleteObjectWithContext(t *testing.T) {
	type fields struct {
		spanCorrelation bool
		mockSetup       func() (expectedReturn interface{})
	}
	type args struct {
		ctx   aws.Context
		input *s3.DeleteObjectInput
		opts  []request.Option
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "instrumentedS3.DeleteObjectWithContext should be delegated to S3.DeleteObjectWithContext while metrics and spans are linked",
			fields: fields{
				spanCorrelation: true,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.DeleteObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.DeleteObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
		{
			name: "instrumentedS3.DeleteObjectWithContext should be delegated to S3.DeleteObjectWithContext while metrics and spans are NOT linked",
			fields: fields{
				spanCorrelation: false,
				mockSetup: func() (expectedReturn interface{}) {
					expectedReturn = &s3.DeleteObjectOutput{}
					return
				},
			},
			args: args{
				ctx: context.Background(),
				input: &s3.DeleteObjectInput{
					Bucket: aws.String(s3bucket),
				},
				opts: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, mockedTracer := mocktrace.NewTracerProviderAndTracer(instrumentationName)
			mockedMeterImp, mockedMeter := mockmetric.NewMeter()
			mockedCounters := createCounters(mockedMeter)
			mockedRecorders := createRecorders(mockedMeter)
			mockedPropagators := global.TextMapPropagator()

			s3Mock := &mocks.MockS3Client{}
			s := &instrumentedS3{
				S3API:           s3Mock,
				tracer:          mockedTracer,
				meter:           mockedMeter,
				propagators:     mockedPropagators,
				counters:        mockedCounters,
				recorders:       mockedRecorders,
				spanCorrelation: tt.fields.spanCorrelation,
			}
			expectedReturn := tt.fields.mockSetup()
			got, err := s.DeleteObjectWithContext(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteObjectWithContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, expectedReturn) {
				t.Errorf("DeleteObjectWithContext() got = %v, want %v", got, expectedReturn)
			}
			spans := mockedTracer.EndedSpans()
			assert.Equal(t, 1, len(spans))
			assert.Equal(t, trace.SpanKindClient, spans[0].Kind)
			assert.Equal(t, s3StorageSystemValue, getLabelValFromSpan(storageSystemKey, *spans[0]).AsString())
			assert.Equal(t, *tt.args.input.Bucket, getLabelValFromSpan(storageDestinationKey, *spans[0]).AsString())
			assert.Equal(t, operationDeleteObject, getLabelValFromSpan(storageOperationKey, *spans[0]).AsString())

			// In Meter we have one duration recorder, one operation counter
			assert.Equal(t, 2, len(mockedMeterImp.MeasurementBatches))

			assertMetrics(t, mockedMeterImp)

			assertSpanCorrelation(t, tt.fields.spanCorrelation, mockedMeterImp, spans[0])
		})
	}
}

func Test_instrumentedS3_NewInstrumentedS3Client(t *testing.T) {
	type args struct {
		s    s3iface.S3API
		opts []Option
	}
	tracerProvider, _ := mocktrace.NewTracerProviderAndTracer(instrumentationName)
	_, meterProvider := mockmetric.NewMeterProvider()
	mockedPropagator := global.TextMapPropagator()
	s3MockClient := &mocks.MockS3Client{}

	tests := []struct {
		name       string
		args       args
		verifyFunc func(t *testing.T, got s3iface.S3API, err error)
	}{
		{
			name: "NewInstrumentedS3Client should return a new client",
			args: args{
				s: s3MockClient,
				opts: []Option{
					WithTracerProvider(tracerProvider),
					WithMeterProvider(meterProvider),
					WithPropagators(mockedPropagator),
					WithSpanCorrelation(true),
				},
			},
			verifyFunc: func(t *testing.T, got s3iface.S3API, err error) {
				assert.Nil(t, err, "error should be nil")
				assert.Equal(t, got.(*instrumentedS3).spanCorrelation, true)
				assert.Equal(t, got.(*instrumentedS3).propagators, mockedPropagator)
				assert.Equal(t, got.(*instrumentedS3).S3API, s3MockClient)
				assert.NotNil(t, got.(*instrumentedS3).meter, "meter should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).tracer, "tracer should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).counters, "counters should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).recorders, "recorders should not be nil")
			},
		},
		{
			name: "NewInstrumentedS3Client with no options should return a new client with default values",
			args: args{
				s:    s3MockClient,
				opts: nil,
			},
			verifyFunc: func(t *testing.T, got s3iface.S3API, err error) {
				assert.Nil(t, err, "error should be nil")
				assert.Equal(t, got.(*instrumentedS3).propagators, global.TextMapPropagator())
				assert.NotNil(t, got.(*instrumentedS3).meter, "meter should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).tracer, "tracer should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).counters, "counters should not be nil")
				assert.NotNil(t, got.(*instrumentedS3).recorders, "recorders should not be nil")
				assert.Equal(t, got.(*instrumentedS3).S3API, s3MockClient)
			},
		},
		{
			name: "NewInstrumentedS3Client with no s3 interface should return error",
			args: args{
				opts: nil,
			},
			verifyFunc: func(t *testing.T, got s3iface.S3API, err error) {
				assert.NotNil(t, err, "error should not be nil")
				assert.Equal(t, err.Error(), "interface must be set")
				assert.Equal(t, got, &instrumentedS3{})
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewInstrumentedS3Client(tt.args.s, tt.args.opts...)
			tt.verifyFunc(t, got, err)
		})
	}
}
