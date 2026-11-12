// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgin_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Restoring the original request instead of just its context must fail the
// outer middleware's PostForm read after ShouldBind has consumed the body.
func TestMiddlewarePreservesBoundFormForOuterMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name      string
		multipart bool
	}{
		{name: "urlencoded"},
		{name: "multipart", multipart: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewBufferString("name=Ada+Lovelace")
			contentType := "application/x-www-form-urlencoded"
			if tc.multipart {
				body.Reset()
				writer := multipart.NewWriter(body)
				require.NoError(t, writer.WriteField("name", "Ada Lovelace"))
				require.NoError(t, writer.Close())
				contentType = writer.FormDataContentType()
			}

			type contextKey struct{}
			originalContext := context.WithValue(t.Context(), contextKey{}, "outer")
			r := httptest.NewRequestWithContext(originalContext, http.MethodPost, "/form?source=query", body)
			r.Header.Set("Content-Type", contentType)
			provider := sdktrace.NewTracerProvider()
			t.Cleanup(func() {
				assert.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
			})

			router := gin.New()
			outerRan := false
			router.Use(func(c *gin.Context) {
				c.Next()
				outerRan = true
				assert.Same(t, originalContext, c.Request.Context())
				assert.Equal(t, "Ada Lovelace", c.PostForm("name"))
				assert.Equal(t, "Ada Lovelace", c.Request.PostForm.Get("name"))
				assert.Equal(t, "Ada Lovelace", c.Request.Form.Get("name"))
				assert.Equal(t, "query", c.Request.Form.Get("source"))
			})
			router.Use(otelgin.Middleware("test",
				otelgin.WithTracerProvider(provider),
				otelgin.WithMeterProvider(metricnoop.NewMeterProvider()),
				otelgin.WithPropagators(propagation.NewCompositeTextMapPropagator()),
			))
			handlerRan := false
			router.POST("/form", func(c *gin.Context) {
				handlerRan = true
				assert.True(t, trace.SpanFromContext(c.Request.Context()).SpanContext().IsValid())
				assert.NotSame(t, originalContext, c.Request.Context())
				var form struct {
					Name string `form:"name"`
				}
				require.NoError(t, c.ShouldBind(&form))
				assert.Equal(t, "Ada Lovelace", form.Name)
				c.Status(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, r)
			assert.True(t, handlerRan)
			assert.True(t, outerRan)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
