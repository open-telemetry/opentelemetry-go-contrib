// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Instrumented provides an example rolldice service that is instrumented with
// OpenTelemetry.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return err
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Start HTTP server.
	port := os.Getenv("APPLICATION_PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		return err
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	return err
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// handleFunc is a replacement for mux.HandleFunc
	// which enriches the handler's HTTP instrumentation with the pattern as the http.route.
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	// Register handlers.
	handleFunc("/rolldice", handleRollDice)

	// Add HTTP instrumentation for the whole server.
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}

func handleRollDice(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	rollsParam := r.URL.Query().Get("rolls")
	player := r.URL.Query().Get("player")

	// Default rolls = 1 if not defined
	if rollsParam == "" {
		rollsParam = "1"
	}

	// Check if rolls is a number
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

	results, err := rollDice(r.Context(), rolls)
	if err != nil {
		// Signals invalid input (<=0)
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
