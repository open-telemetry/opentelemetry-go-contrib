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

func handleRolldice(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
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

	if len(results) == 1 {
		writeJSON(w, results[0])
	} else {
		writeJSON(w, results)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Internal Server Error",
		})
		log.Printf("ERROR: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func rolldice(rolls int) ([]int, error) {
	const maxRolls = 1000 // Arbitrary limit to prevent Slice memory allocation with excessive size value.

	if rolls > maxRolls {
		return nil, errors.New("rolls parameter exceeds maximum allowed value")
	}

	if rolls <= 0 {
		return nil, errors.New("rolls must be positive")
	}

	if rolls == 1 {
		return []int{rollOnce()}, nil
	}

	results := make([]int, rolls)
	for i := range rolls {
		results[i] = rollOnce()
	}
	return results, nil
}

// rollOnce returns a random number between 1 and 6.
func rollOnce() int {
	roll := 1 + rand.Intn(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	return roll
}
