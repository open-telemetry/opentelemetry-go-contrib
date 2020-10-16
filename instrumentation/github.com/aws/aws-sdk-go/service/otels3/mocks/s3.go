package mocks

import (
	//mockmeter "go.opentelemetry.io/contrib/internal/metric"
	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	"go.opentelemetry.io/otel/api/trace"
)

type S3Client struct {
	S3API
}

type MockTracerProvider struct {
	tracer *mocktrace.Tracer
}

func (m *MockTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return m.tracer
}

func NewTracerProvider() *MockTracerProvider {
	return &MockTracerProvider{
		tracer: mocktrace.NewTracer("github.com/aws/aws-sdk-go/aws/service/s3"),
	}
}
