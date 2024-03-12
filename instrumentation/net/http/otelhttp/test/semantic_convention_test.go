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

package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type lc struct {
	t testing.TB
}

func (l *lc) Accept(log testcontainers.Log) {
	l.t.Log(string(log.Content))
}

func TestSemanticConventions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping tests of the semantic conventions in short mode.")
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/madvikinggod/semantic-convention-checker:0.0.11",
		ExposedPorts: []string{"4318/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("4318/tcp"),
			wait.ForLog("INFO starting server address="),
		),

		Mounts: []testcontainers.ContainerMount{
			{
				Source: testcontainers.GenericBindMountSource{
					HostPath: fmt.Sprintf("%s/config.yaml", wd),
				},
				Target: "/config.yaml",
			},
		},
	}

	scc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Logger:           testcontainers.TestLogger(t),
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, scc.Terminate(ctx))
	}()
	scc.FollowOutput(&lc{t: t})
	err = scc.StartLogProducer(ctx)
	require.NoError(t, err)

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		t.Errorf("Missing attributes: %v", err)
		rc, err := scc.Logs(context.Background())
		if err != nil {
			return
		}
		stdout, err := io.ReadAll(rc)
		if err != nil {
			return
		}
		t.Log(stdout)
	}))

	endpoint, err := scc.Endpoint(ctx, "")
	assert.NoError(t, err)

	t.Log(endpoint)

	tp, err := newTraceProvider(endpoint)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("/", otelhttp.NewHandler(http.HandlerFunc(hello), "http.server.root", otelhttp.WithTracerProvider(tp)))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := ts.Client()
	client.Transport = otelhttp.NewTransport(client.Transport, otelhttp.WithSpanNameFormatter(formatter), otelhttp.WithTracerProvider(tp))

	resp, err := client.Post(ts.URL+"/", "application/text", strings.NewReader("Hello, Server!"))
	require.NoError(t, err)

	_, err = io.Copy(io.Discard, resp.Body)
	assert.NoError(t, err)
	err = resp.Body.Close()
	assert.NoError(t, err)

	err = tp.ForceFlush(context.Background())
	require.NoError(t, err)

	time.Sleep(5 * time.Second)
}

func newTraceProvider(url string) (*trace.TracerProvider, error) {
	res := resource.NewSchemaless()

	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(url),
	)
	// traceExporter, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)
	return traceProvider, nil
}

func formatter(operation string, r *http.Request) string {
	return strings.Join([]string{"http.client", r.Method, operation}, ".")
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello World")
}
