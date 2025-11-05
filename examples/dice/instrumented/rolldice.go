// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync/atomic"

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
	lastRolls      atomic.Int64
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

	// Register the gauge callback.
	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(lastRollsGauge, lastRolls.Load())
			return nil
		},
		lastRollsGauge,
	)
	if err != nil {
		panic(err)
	}
}

func handleRolldice(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters.
	rollsParam := r.URL.Query().Get("rolls")
	player := r.URL.Query().Get("player")
	maxRolls := 1000 // Arbitrary limit to prevent Slice memory allocation with excessive size value.

	// Default rolls = 1 if not defined.
	if rollsParam == "" {
		rollsParam = "1"
	}

	// Check if rolls is a number.
	rolls, err := strconv.Atoi(rollsParam)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		msg := "Parameter rolls must be a positive integer"
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": msg,
		})
		logger.WarnContext(r.Context(), msg)
		return
	}

	results, err := rollDice(r.Context(), rolls)
	if err != nil || rolls > maxRolls {
		// Signals invalid input (<=0).
		w.WriteHeader(http.StatusInternalServerError)
		logger.ErrorContext(r.Context(), err.Error())
		return
	}

	if player == "" {
		logger.DebugContext(r.Context(), "anonymous player rolled", "results", results)
	} else {
		logger.DebugContext(r.Context(), "player", player, "results", results)
	}
	logger.InfoContext(r.Context(), "Some player rolled a dice.")

	w.Header().Set("Content-Type", "application/json")
	if len(results) == 1 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		json.NewEncoder(w).Encode(results)
	}
}

// rollDice is the outer function which Does the error handling.
func rollDice(ctx context.Context, rolls int) ([]int, error) {
	ctx, span := tracer.Start(ctx, "rollDice")
	defer span.End()

	if rolls <= 0 {
		err := errors.New("rolls must be positive")
		span.RecordError(err)
		return nil, err
	}

	results := make([]int, rolls)
	for i := 0; i < rolls; i++ {
		results[i] = rollOnce(ctx)
		outcomeHist.Record(ctx, int64(results[i]))
	}

	rollsAttr := attribute.Int("rolls", rolls)
	span.SetAttributes(rollsAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollsAttr))
	lastRolls.Store(int64(rolls))
	return results, nil
}

// rollOnce is the inner function — returns a random number 1–6.
func rollOnce(ctx context.Context) int {
	ctx, span := tracer.Start(ctx, "rollOnce")
	defer span.End()

	roll := 1 + rand.IntN(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)

	return roll
}
