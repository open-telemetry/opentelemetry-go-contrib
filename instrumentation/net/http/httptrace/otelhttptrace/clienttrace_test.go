// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttptrace

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func ExampleNewClientTrace() {
	client := http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return NewClientTrace(ctx)
			}),
		),
	}

	resp, err := client.Get("https://example.com")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(resp.Status)
}

type zeroTripper struct{}

func (zeroTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

var _ http.RoundTripper = zeroTripper{}

// TestNewClientParallelismWithoutSubspans tests running many Gets on a client simultaneously,
// which would trigger a race condition if root were not protected by a mutex.
func TestNewClientParallelismWithoutSubspans(t *testing.T) {
	t.Parallel()

	makeClientTrace := func(ctx context.Context) *httptrace.ClientTrace {
		return NewClientTrace(ctx, WithoutSubSpans())
	}

	client := http.Client{
		Transport: otelhttp.NewTransport(
			zeroTripper{},
			otelhttp.WithClientTrace(makeClientTrace),
		),
	}

	var wg sync.WaitGroup

	for i := 1; i < 10000; i++ {
		wg.Add(1)
		go func() {
			resp, err := client.Get("}}}}}")
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
			wg.Done()
		}()
	}

	wg.Wait()
}
