package fluentforward

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	url         = "localhost:24224"
	serviceName = "fluentforward"
)

func TestInstallNewPipeline(t *testing.T) {
	instance := startMockFluentServer(t)
	defer instance.Close()

	err := InstallNewPipeline(url, serviceName)
	assert.NoError(t, err)
}

func TestNewExportPipeline(t *testing.T) {
	instance := startMockFluentServer(t)
	defer instance.Close()

	testCases := []struct {
		name                                  string
		options                               []Option
		testSpanSampling, spanShouldBeSampled bool
	}{
		{
			name: "simple pipeline",
		},

		{
			name: "always on",
			options: []Option{
				WithSDK(&sdktrace.Config{
					DefaultSampler: sdktrace.AlwaysSample(),
				}),
			},
			testSpanSampling:    true,
			spanShouldBeSampled: true,
		},

		{
			name: "never",
			options: []Option{
				WithSDK(&sdktrace.Config{
					DefaultSampler: sdktrace.NeverSample(),
				}),
			},
			testSpanSampling:    true,
			spanShouldBeSampled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tp, err := NewExportPipeline(
				url,
				serviceName,
				tc.options...,
			)
			assert.NoError(t, err)
			assert.NotEqual(t, tp, sdktrace.TracerProvider{})

			if tc.testSpanSampling {
				_, span := tp.Tracer("fluentforward test").Start(context.Background(), tc.name)
				spanCtx := span.SpanContext()
				assert.Equal(t, tc.spanShouldBeSampled, spanCtx.IsSampled())
				span.End()
			}
		})
	}
}

func TestNewRawExporter(t *testing.T) {
	instance := startMockFluentServer(t)
	defer instance.Close()

	exp, err := NewRawExporter(
		url,
		serviceName,
	)

	assert.NoError(t, err)
	assert.EqualValues(t, serviceName, exp.serviceName)
}

func TestNewRawExporterShouldFailInvalidURL(t *testing.T) {
	exp, err := NewRawExporter("", serviceName)
	assert.Error(t, err)
	assert.EqualError(t, err, "fluent instance url cannot be empty")
	assert.Nil(t, exp)
}

func TestExportSpans(t *testing.T) {
	instance := startMockFluentServer(t)
	defer instance.Close()

	spans := []*export.SpanData{
		{
			SpanContext: trace.SpanContext{
				TraceID: trace.TraceID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
				SpanID:  trace.SpanID{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8},
			},
			ParentSpanID:  trace.SpanID{},
			SpanKind:      trace.SpanKindServer,
			Name:          "foo",
			StartTime:     time.Date(2020, time.November, 14, 00, 24, 0, 0, time.UTC),
			EndTime:       time.Date(2020, time.November, 14, 00, 25, 0, 0, time.UTC),
			Attributes:    nil,
			MessageEvents: nil,
			StatusCode:    codes.Error,
		},
	}

	exp, err := NewRawExporter(url, serviceName)
	assert.NoError(t, err)
	ctx := context.Background()

	err = exp.ExportSpans(ctx, spans)
	assert.NoError(t, err)
}

type mockFluentServer struct {
	t      *testing.T
	wg     *sync.WaitGroup
	server *http.Server
}

func (f *mockFluentServer) handler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	require.NoError(f.t, err)
	require.NotNil(f.t, data)
	w.WriteHeader(http.StatusAccepted)
}

func startMockFluentServer(t *testing.T) *mockFluentServer {
	instance := &mockFluentServer{
		t: t,
	}
	listener, err := net.Listen("tcp", "127.0.0.1:24224")
	require.NoError(t, err)

	server := &http.Server{
		Handler: http.HandlerFunc(instance.handler),
	}
	instance.server = server

	wg := &sync.WaitGroup{}
	wg.Add(1)
	instance.wg = wg
	go func() {
		err := server.Serve(listener)
		require.Equal(t, http.ErrServerClosed, err)
		wg.Done()
	}()

	return instance
}

func (f *mockFluentServer) Close() {
	server := f.server
	f.server = nil
	require.NoError(f.t, server.Shutdown(context.Background()))
	f.wg.Wait()
}
