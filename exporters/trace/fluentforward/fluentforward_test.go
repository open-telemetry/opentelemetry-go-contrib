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

package fluentforward

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/global"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
			assert.NotEqual(t, tp, global.TracerProvider())

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

type mockFluentServer struct {
	t      *testing.T
	wg     *sync.WaitGroup
	server *http.Server
}

func (f *mockFluentServer) handler(w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(r.Body)
	require.NoError(f.t, err)
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
