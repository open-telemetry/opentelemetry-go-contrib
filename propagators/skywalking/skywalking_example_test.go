// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking_test

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/propagators/skywalking"
)

func ExampleSkywalking() {
	// Create a new SkyWalking propagator
	skyWalkingPropagator := skywalking.Skywalking{}

	// Set up the propagator in the global provider
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			skyWalkingPropagator,
			propagation.TraceContext{}, // Also support W3C trace context
			propagation.Baggage{},      // Also support baggage
		),
	)

	// Create a span context to propagate
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		log.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		log.Fatal(err)
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	// Create a context with the span context
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	// Inject the context into a carrier (e.g., HTTP headers)
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	fmt.Printf("SkyWalking header set: %t\n", carrier.Get("sw8") != "")

	// Extract the context from the carrier
	extractedCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	extractedSC := trace.SpanContextFromContext(extractedCtx)

	fmt.Printf("Context extracted successfully: %t\n", extractedSC.IsValid())
	fmt.Printf("Trace ID preserved: %t\n", extractedSC.TraceID() == traceID)

	// Output:
	// SkyWalking header set: true
	// Context extracted successfully: true
	// Trace ID preserved: true
}

func ExampleSkywalking_correlation() {
	// Create a new SkyWalking propagator
	skyWalkingPropagator := skywalking.Skywalking{}

	// Create correlation data using baggage
	member1, _ := baggage.NewMember("service.name", "web-service")
	member2, _ := baggage.NewMember("user.id", "user123")
	member3, _ := baggage.NewMember("request.type", "api")

	bags, err := baggage.New(member1, member2, member3)
	if err != nil {
		log.Fatal(err)
	}

	// Create a span context to propagate
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		log.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		log.Fatal(err)
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	// Create a context with the span context and baggage
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	// Inject the context into a carrier (e.g., HTTP headers)
	carrier := make(propagation.MapCarrier)
	skyWalkingPropagator.Inject(ctx, carrier)

	fmt.Printf("SW8 header set: %t\n", carrier.Get("sw8") != "")
	fmt.Printf("SW8-Correlation header set: %t\n", carrier.Get("sw8-correlation") != "")

	// Extract the context from the carrier
	extractedCtx := skyWalkingPropagator.Extract(context.Background(), carrier)
	extractedSC := trace.SpanContextFromContext(extractedCtx)
	extractedBags := baggage.FromContext(extractedCtx)

	fmt.Printf("Context extracted successfully: %t\n", extractedSC.IsValid())
	fmt.Printf("Correlation data preserved: %t\n", extractedBags.Len() == 3)
	fmt.Printf("Service name: %s\n", extractedBags.Member("service.name").Value())

	// Output:
	// SW8 header set: true
	// SW8-Correlation header set: true
	// Context extracted successfully: true
	// Correlation data preserved: true
	// Service name: web-service
}
