// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"math/rand"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

const name = "go.opentelemetry.io/contrib/examples/dice"

var (
	tracer         = otel.Tracer(name)
	meter          = otel.Meter(name)
	logger         = otelslog.NewLogger(name)
	rollCnt        metric.Int64Counter
	outcomeHist    metric.Int64Histogram
	lastRollsGauge metric.Int64ObservableGauge
	lastRolls      int64
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls"),
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
		metric.WithDescription("The last rolls value observed"),
	)
	if err != nil {
		panic(err)
	}

	// Register the gauge callback
	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(lastRollsGauge, lastRolls)
			return nil
		},
		lastRollsGauge,
	)
	if err != nil {
		panic(err)
	}
}

// rollDice is the outer function which Does the error handling.
func rollDice(ctx context.Context, rolls int) ([]int, error) {
	ctx, span := tracer.Start(ctx, "rollDice")
	defer span.End()

	if rolls <= 0 {
		err := errors.New("rolls must be positive")
		span.RecordError(err)
		logger.ErrorContext(ctx, "error", "error", err)
		return nil, err
	}

	if rolls == 1 {
		val := rollOnce(ctx)
		outcomeHist.Record(ctx, int64(val))
		lastRolls = int64(rolls)
		return []int{val}, nil
	}

	results := make([]int, rolls)
	for i := 0; i < rolls; i++ {
		results[i] = rollOnce(ctx)
		outcomeHist.Record(ctx, int64(results[i]))
	}

	rollsAttr := attribute.Int("rolls", rolls)
	span.SetAttributes(rollsAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollsAttr))
	lastRolls = int64(rolls)
	return results, nil
}

// rollOnce is the inner function — returns a random number 1–6.
func rollOnce(ctx context.Context) int {
	ctx, span := tracer.Start(ctx, "rollOnce")
	defer span.End()

	roll := 1 + rand.Intn(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)

	return roll
}
