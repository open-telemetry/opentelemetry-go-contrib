// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xrayconfig

// Pared down version of go.opentelemtry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/mock_collector_test.go
// for end to end testing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func makeMockCollector(t *testing.T, mockConfig *mockConfig) *mockCollector {
	return &mockCollector{
		t: t,
		traceSvc: &mockTraceService{
			storage: NewSpansStorage(),
			errors:  mockConfig.errors,
		},
	}
}

type mockTraceService struct {
	collectortracepb.UnimplementedTraceServiceServer

	errors   []error
	requests int
	mu       sync.RWMutex
	storage  SpansStorage
	headers  metadata.MD
	delay    time.Duration
}

func (mts *mockTraceService) getResourceSpans() []*tracepb.ResourceSpans {
	mts.mu.RLock()
	defer mts.mu.RUnlock()
	return mts.storage.GetResourceSpans()
}

func (mts *mockTraceService) Export(ctx context.Context, exp *collectortracepb.ExportTraceServiceRequest) (*collectortracepb.ExportTraceServiceResponse, error) {
	if mts.delay > 0 {
		time.Sleep(mts.delay)
	}

	mts.mu.Lock()
	defer func() {
		mts.requests++
		mts.mu.Unlock()
	}()

	reply := &collectortracepb.ExportTraceServiceResponse{}
	if mts.requests < len(mts.errors) {
		idx := mts.requests
		return reply, mts.errors[idx]
	}

	mts.headers, _ = metadata.FromIncomingContext(ctx)
	mts.storage.AddSpans(exp)
	return reply, nil
}

type mockCollector struct {
	t *testing.T

	traceSvc *mockTraceService

	endpoint string
	ln       *listener
	stopFunc func()
	stopOnce sync.Once
}

type mockConfig struct {
	errors   []error
	endpoint string
}

var _ collectortracepb.TraceServiceServer = (*mockTraceService)(nil)

var errAlreadyStopped = fmt.Errorf("already stopped")

func (mc *mockCollector) stop() error {
	err := errAlreadyStopped
	mc.stopOnce.Do(func() {
		err = nil
		if mc.stopFunc != nil {
			mc.stopFunc()
		}
	})
	// Give it sometime to shutdown.
	<-time.After(160 * time.Millisecond)

	// Getting the lock ensures the traceSvc is done flushing.
	mc.traceSvc.mu.Lock()
	defer mc.traceSvc.mu.Unlock()

	return err
}

func (mc *mockCollector) Stop() error {
	return mc.stop()
}

func (mc *mockCollector) getResourceSpans() []*tracepb.ResourceSpans {
	return mc.traceSvc.getResourceSpans()
}

func (mc *mockCollector) GetResourceSpans() []*tracepb.ResourceSpans {
	return mc.getResourceSpans()
}

func runMockCollectorAtEndpoint(t *testing.T, endpoint string) *mockCollector {
	return runMockCollectorWithConfig(t, &mockConfig{endpoint: endpoint})
}

func runMockCollectorWithConfig(t *testing.T, mockConfig *mockConfig) *mockCollector {
	ln, err := net.Listen("tcp", mockConfig.endpoint)
	if err != nil {
		t.Fatalf("Failed to get an endpoint: %v", err)
	}

	srv := grpc.NewServer()
	mc := makeMockCollector(t, mockConfig)
	collectortracepb.RegisterTraceServiceServer(srv, mc.traceSvc)
	mc.ln = newListener(ln)
	go func() {
		_ = srv.Serve((net.Listener)(mc.ln))
	}()

	mc.endpoint = ln.Addr().String()
	// srv.Stop calls Close on mc.ln.
	mc.stopFunc = srv.Stop

	return mc
}

type listener struct {
	closeOnce sync.Once
	wrapped   net.Listener
	C         chan struct{}
}

func newListener(wrapped net.Listener) *listener {
	return &listener{
		wrapped: wrapped,
		C:       make(chan struct{}, 1),
	}
}

func (l *listener) Close() error { return l.wrapped.Close() }

func (l *listener) Addr() net.Addr { return l.wrapped.Addr() }

// Accept waits for and returns the next connection to the listener. It will
// send a signal on l.C that a connection has been made before returning.
func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			// If the listener has been closed, do not allow callers of
			// WaitForConn to wait for a connection that will never come.
			l.closeOnce.Do(func() { close(l.C) })
		}
		return conn, err
	}

	select {
	case l.C <- struct{}{}:
	default:
		// If C is full, assume nobody is listening and move on.
	}
	return conn, nil
}

// WaitForConn will wait indefintely for a connection to be established with
// the listener before returning.
func (l *listener) WaitForConn() {
	for {
		select {
		case <-l.C:
			return
		default:
			runtime.Gosched()
		}
	}
}
