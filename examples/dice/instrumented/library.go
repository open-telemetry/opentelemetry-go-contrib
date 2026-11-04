// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

const name = "go.opentelemetry.io/contrib/examples/dice"

var (
	tracer = otel.Tracer(name)
	meter  = otel.Meter(name)

	rollCnt        metric.Int64Counter
	outcomeHist    metric.Int64Histogram
	lastRollsGauge metric.Int64ObservableGauge
	lastRolls      atomic.Int64
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("Number of dice rolled"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}

	outcomeHist, err = meter.Int64Histogram(
		"dice.outcome",
		metric.WithDescription("Distribution of dice outcomes (1-6)"),
		metric.WithUnit("{count}"),
	)
	if err != nil {
		panic(err)
	}

	lastRollsGauge, err = meter.Int64ObservableGauge(
		"dice.last.rolls",
		metric.WithDescription("The most recent rolled value"),
	)
	if err != nil {
		panic(err)
	}

	// Register the gauge callback.
	_, err = meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			o.ObserveInt64(lastRollsGauge, lastRolls.Load())
			return nil
		},
		lastRollsGauge,
	)
	if err != nil {
		panic(err)
	}
}

func rollDice(ctx context.Context, rolls int) ([]int, error) {
	const maxRolls = 1000 // Arbitrary limit to prevent Slice memory allocation with excessive size value.

	ctx, span := tracer.Start(ctx, "rollDice")
	defer span.End()

	rollsAttr := attribute.Int("rolls", rolls)
	span.SetAttributes(rollsAttr)

	if rolls > maxRolls {
		err := errors.New("rolls parameter exceeds maximum allowed value")
		span.RecordError(err) //nolint:forbidigo // RecordError is used here to demonstrate error recording in spans as required by the specification.
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if rolls <= 0 {
		err := errors.New("rolls must be positive")
		span.RecordError(err) //nolint:forbidigo // RecordError is used here to demonstrate error recording in spans as required by the specification.
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	results := make([]int, rolls)
	for i := range results {
		results[i] = rollOnce(ctx)
		outcomeHist.Record(ctx, int64(results[i]))
	}

	rollCnt.Add(ctx, int64(rolls), metric.WithAttributes(rollsAttr))
	lastRolls.Store(int64(rolls))
	return results, nil
}

// rollOnce returns a random number between 1 and 6.
func rollOnce(ctx context.Context) int {
	_, span := tracer.Start(ctx, "rollOnce")
	defer span.End()

	roll := 1 + rand.IntN(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)

	return roll
}
