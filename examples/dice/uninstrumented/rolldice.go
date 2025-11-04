// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

func handleRollDice(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters.
	rollsParam := r.URL.Query().Get("rolls")
	player := r.URL.Query().Get("player")

	// Default rolls = 1 if not defined.
	if rollsParam == "" {
		rollsParam = "1"
	}

	// Check if rolls is a number.
	rolls, err := strconv.Atoi(rollsParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := "Parameter rolls must be a positive integer"
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": msg,
		})
		log.Printf("WARN: %s", msg)
		return
	}

	results, err := rolldice(rolls)
	if err != nil {
		// Signals invalid input (<=0).
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %v", err)
		return
	}

	if player == "" {
		log.Printf("DEBUG: anonymous player rolled %v", results)
	} else {
		log.Printf("DEBUG: player=%s rolled %v", player, results)
	}
	log.Printf("INFO: %s %s -> 200 OK", r.Method, r.URL.String())

	w.Header().Set("Content-Type", "application/json")
	if len(results) == 1 {
		json.NewEncoder(w).Encode(results[0])
	} else {
		json.NewEncoder(w).Encode(results)
	}
}

// rolldice is the outer function which does the error handling.
func rolldice(rolls int) ([]int, error) {
	if rolls <= 0 {
		return nil, errors.New("rolls must be positive")
	}

	if rolls == 1 {
		return []int{rollOnce()}, nil
	}

	results := make([]int, rolls)
	for i := 0; i < rolls; i++ {
		results[i] = rollOnce()
	}
	return results, nil
}

// rollOnce is the inner function — returns a random number 1–6.
func rollOnce() int {
	roll := 1 + rand.Intn(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	return roll
}
