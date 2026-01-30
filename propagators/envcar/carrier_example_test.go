// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/propagators/envcar"
)

// This example is a go program where the environment variables are carrying the
// trace information, and we're going to pick them up into our context.
func ExampleCarrier_extractFromParent() {
	// Simulate environment variables set by a parent process.
	// In practice, these would already be set when this process starts.
	_ = os.Setenv("TRACEPARENT", "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01")

	// Create a carrier to read trace context from environment variables.
	carrier := envcar.Carrier{}

	// Extract trace context that was propagated by the parent process.
	prop := propagation.TraceContext{}
	ctx := prop.Extract(context.Background(), carrier)

	// The context now contains the span context from the parent.
	spanCtx := trace.SpanContextFromContext(ctx)
	fmt.Printf("Trace ID: %s\n", spanCtx.TraceID())
	fmt.Printf("Span ID: %s\n", spanCtx.SpanID())
	fmt.Printf("Sampled: %t\n", spanCtx.IsSampled())
	// Output:
	// Trace ID: 0102030405060708090a0b0c0d0e0f10
	// Span ID: 0102030405060708
	// Sampled: true
}

// This example is a go program where we have a trace and we'd like to inject it
// into a command we're going to run.
func ExampleCarrier_childProcess() {
	// Create a span context with a known trace ID.
	traceID := trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Prepare a command that prints the TRACEPARENT environment variable.
	cmd := exec.Command("printenv", "TRACEPARENT")
	cmd.Env = os.Environ()

	// Create a carrier that injects trace context into the child
	// process's environment rather than the current process's.
	carrier := envcar.Carrier{
		SetEnvFunc: func(key, value string) {
			cmd.Env = append(cmd.Env, key+"="+value)
		},
	}

	// Inject trace context into the child's environment.
	prop := propagation.TraceContext{}
	prop.Inject(ctx, carrier)

	// The child process now has trace context in its environment,
	// independent of the parent process's environment variables.
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Print(string(out))
	// Output: 00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01
}
