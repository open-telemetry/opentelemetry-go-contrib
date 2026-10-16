// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func BenchmarkMiddlewareContextRestore(b *testing.B) {
	router := gin.New()
	router.Use(otelgin.Middleware("benchmark",
		otelgin.WithTracerProvider(noop.NewTracerProvider()),
		otelgin.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelgin.WithPropagators(propagation.NewCompositeTextMapPropagator()),
	))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	request := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/", http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
	}
}
