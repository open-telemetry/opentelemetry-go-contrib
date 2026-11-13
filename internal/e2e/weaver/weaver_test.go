// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package weaver_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	weaverImage = "otel/weaver"
	weaverTag   = "v0.23.0"
)

// TestWeaverLiveCheck spins up a weaver container via dockertest, exercises
// otelhttp instrumentation against it, and validates the resulting
// live-check report.
func TestWeaverLiveCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping weaver live-check in short mode")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Skipf("skipping: docker not available: %v", err)
	}
	if err := pool.Client.Ping(); err != nil {
		t.Skipf("skipping: docker daemon not reachable: %v", err)
	}
	pool.MaxWait = 2 * time.Minute

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: weaverImage,
		Tag:        weaverTag,
		Cmd: []string{
			"registry", "live-check",
			"--format", "json",
			"--output", "/reports",
			"--inactivity-timeout", "30",
		},
		ExposedPorts: []string{"4317/tcp"},
	}, func(config *docker.HostConfig) {
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("start weaver container: %v", err)
	}
	t.Cleanup(func() {
		if purgeErr := pool.Purge(resource); purgeErr != nil {
			t.Logf("purge weaver container: %v", purgeErr)
		}
	})

	otlpEndpoint := fmt.Sprintf("localhost:%s", resource.GetPort("4317/tcp"))
	t.Logf("weaver OTLP endpoint: %s", otlpEndpoint)

	// Wait for the OTLP gRPC listener inside the container to accept connections.
	if err := pool.Retry(func() error {
		dialer := &net.Dialer{Timeout: 2 * time.Second}
		conn, dialErr := dialer.DialContext(t.Context(), "tcp", otlpEndpoint)
		if dialErr != nil {
			return dialErr
		}
		return conn.Close()
	}); err != nil {
		t.Fatalf("weaver OTLP listener not ready: %v", err)
	}

	ctx := t.Context()

	shutdown, err := initOTLP(ctx, otlpEndpoint)
	if err != nil {
		t.Fatalf("init OTLP: %v", err)
	}
	defer func() {
		if shutdown == nil {
			return
		}
		if shutdownErr := shutdown(ctx); shutdownErr != nil {
			t.Logf("shutdown OTLP: %v", shutdownErr)
		}
	}()

	exerciseOtelHTTP(t, ctx)

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown OTLP: %v", err)
	}
	shutdown = nil

	exitCode, err := waitForWeaver(ctx, pool, resource.Container.ID)
	if err != nil {
		t.Fatalf("wait for weaver live-check: %v", err)
	}
	t.Logf("weaver live-check exit code: %d (warn-only)", exitCode)

	reports, err := downloadReports(ctx, pool, resource.Container.ID)
	if err != nil {
		t.Fatalf("download weaver reports: %v", err)
	}
	if len(reports) == 0 {
		t.Fatal("weaver did not produce any JSON reports")
	}
	logReports(t, reports)
}

// initOTLP configures the global TracerProvider and MeterProvider with
// OTLP gRPC exporters pointing at the given endpoint.
func initOTLP(ctx context.Context, endpoint string) (func(context.Context) error, error) {
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExp))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)
	otel.SetMeterProvider(mp)

	return func(c context.Context) error {
		tpErr := tp.Shutdown(c)
		mpErr := mp.Shutdown(c)
		return errors.Join(
			errWithLabel("trace provider", tpErr),
			errWithLabel("metric provider", mpErr),
		)
	}, nil
}

func errWithLabel(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", label, err)
}

func waitForWeaver(ctx context.Context, pool *dockertest.Pool, containerID string) (int, error) {
	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	exitCode, err := pool.Client.WaitContainerWithContext(containerID, waitCtx)
	if err == nil {
		return exitCode, nil
	}
	if waitCtx.Err() == nil {
		return 0, err
	}

	// Give weaver a chance to finish and write reports before forcing cleanup.
	hupCtx, hupCancel := context.WithTimeout(ctx, 5*time.Second)
	defer hupCancel()
	if killErr := pool.Client.KillContainer(docker.KillContainerOptions{
		ID:      containerID,
		Signal:  docker.SIGHUP,
		Context: hupCtx,
	}); killErr != nil {
		return 0, fmt.Errorf("timed out waiting for inactivity shutdown; send SIGHUP: %w", killErr)
	}

	finalCtx, finalCancel := context.WithTimeout(ctx, 15*time.Second)
	defer finalCancel()
	exitCode, err = pool.Client.WaitContainerWithContext(containerID, finalCtx)
	if err != nil {
		return 0, fmt.Errorf("timed out waiting for inactivity shutdown; wait after SIGHUP: %w", err)
	}
	return exitCode, nil
}

func downloadReports(ctx context.Context, pool *dockertest.Pool, containerID string) (map[string]string, error) {
	var archive bytes.Buffer
	if err := pool.Client.DownloadFromContainer(containerID, docker.DownloadFromContainerOptions{
		OutputStream: &archive,
		Path:         "/reports",
		Context:      ctx,
	}); err != nil {
		return nil, err
	}

	reports := make(map[string]string)
	tr := tar.NewReader(&archive)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.FileInfo().IsDir() || filepath.Ext(header.Name) != ".json" {
			continue
		}
		var b strings.Builder
		if _, err := io.Copy(&b, tr); err != nil {
			return nil, err
		}
		reports[header.Name] = b.String()
	}
	return reports, nil
}

func logReports(t *testing.T, reports map[string]string) {
	t.Helper()

	names := make([]string, 0, len(reports))
	for name := range reports {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		report := reports[name]
		t.Logf("weaver report %s:\n%s", name, report)
	}
}

// exerciseOtelHTTP spins up a local test server wrapped with otelhttp
// and issues requests through an instrumented client transport.
func exerciseOtelHTTP(t *testing.T, ctx context.Context) {
	t.Helper()

	handler := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
		"test-server",
	)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req, err := http.NewRequestWithContext(ctx, method, srv.URL+"/test", http.NoBody)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: %d", resp.StatusCode)
		}
	}
}
