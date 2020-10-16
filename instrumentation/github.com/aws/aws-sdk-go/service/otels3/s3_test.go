package otels3

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go/service/otels3/mocks"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"reflect"
	"testing"
)

var (
	mockedMeter = global.MeterProvider().Meter(
		"github.com/aws/aws-sdk-go/aws/service/s3",
	)
	mockedTracer      = mocks.NewTracerProvider().Tracer("github.com/aws/aws-sdk-go/aws/service/s3")
	mockedCounters    = createCounters(mockedMeter)
	mockedRecorders   = createRecorders(mockedMeter)
	mockedPropagators = global.TextMapPropagator()
)

func Test_instrumentedS3_PutObjectWithContext(t *testing.T) {
	type fields struct {
		S3API                    s3iface.S3API
		tracer                   trace.Tracer
		meter                    metric.Meter
		spanCorrelationInMetrics bool
		mockSetup                func(s3Client *mock.Mock) interface{}
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
			name: "instrumentedS3.PutObjectWithContext should be delegated to S3.PutObjectWithContext",
			fields: fields{
				S3API:                    &mocks.S3Client{},
				spanCorrelationInMetrics: false,
				mockSetup: func(m *mock.Mock) (expectedReturn interface{}) {
					expectedReturn = &s3.PutObjectOutput{}
					m.On("PutObjectWithContext", mock.Anything, mock.Anything).Return(expectedReturn, nil)
					return
				},
			},
			args: args{
				ctx:   context.Background(),
				input: &s3.PutObjectInput{},
				opts:  nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3Mock := &mocks.S3Client{}
			s := &instrumentedS3{
				S3API:                    s3Mock,
				tracer:                   mockedTracer,
				meter:                    mockedMeter,
				propagators:              mockedPropagators,
				counters:                 mockedCounters,
				recorders:                mockedRecorders,
				spanCorrelationInMetrics: tt.fields.spanCorrelationInMetrics,
			}
			expectedReturn := tt.fields.mockSetup(&s3Mock.S3API.Mock)
			got, err := s.PutObjectWithContext(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutObjectWithContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, expectedReturn) {
				t.Errorf("PutObjectWithContext() got = %v, want %v", got, expectedReturn)
			}
		})
	}
}
