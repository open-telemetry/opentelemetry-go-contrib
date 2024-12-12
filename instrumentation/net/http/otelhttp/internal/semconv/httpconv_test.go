// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func TestNewRecordMetrics(t *testing.T) {
	tests := []struct {
		name         string
		setEnv       bool
		expectedFunc func(server HTTPServer, t *testing.T)
	}{
		{
			name:   "set env",
			setEnv: true,
			expectedFunc: func(server HTTPServer, t *testing.T) {
				assert.Equal(t, int64(100), server.requestBodySizeHistogram.(*testRecorder[int64]).value)
				assert.Equal(t, int64(200), server.responseBodySizeHistogram.(*testRecorder[int64]).value)
				assert.Equal(t, float64(300), server.requestDurationHistogram.(*testRecorder[float64]).value)

				want := []attribute.KeyValue{
					attribute.String("url.scheme", "http"),
					attribute.String("http.request.method", "POST"),
					attribute.Int64("http.response.status_code", 301),
					attribute.String("key", "value"),
					attribute.String("server.address", "stuff"),
					attribute.String("network.protocol.name", "http"),
					attribute.String("network.protocol.version", "1.1"),
				}

				assert.ElementsMatch(t, want, server.requestBodySizeHistogram.(*testRecorder[int64]).attributes)
				assert.ElementsMatch(t, want, server.responseBodySizeHistogram.(*testRecorder[int64]).attributes)
				assert.ElementsMatch(t, want, server.requestDurationHistogram.(*testRecorder[float64]).attributes)
			},
		},
		{
			name:   "not set env",
			setEnv: false,
			expectedFunc: func(server HTTPServer, t *testing.T) {
				assert.Equal(t, int64(0), server.requestBodySizeHistogram.(*testRecorder[int64]).value)
				assert.Equal(t, int64(0), server.responseBodySizeHistogram.(*testRecorder[int64]).value)
				assert.Equal(t, float64(0), server.requestDurationHistogram.(*testRecorder[float64]).value)

				assert.Empty(t, server.requestBodySizeHistogram.(*testRecorder[int64]).attributes)
				assert.Empty(t, server.responseBodySizeHistogram.(*testRecorder[int64]).attributes)
				assert.Empty(t, server.requestDurationHistogram.(*testRecorder[float64]).attributes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(OTelSemConvStabilityOptIn, "http/dup")
			}

			server := NewTestHTTPServer()
			req, err := http.NewRequest("POST", "http://example.com", nil)
			assert.NoError(t, err)

			server.RecordMetrics(context.Background(), ServerMetricData{
				ServerName:   "stuff",
				ResponseSize: 200,
				MetricAttributes: MetricAttributes{
					Req:        req,
					StatusCode: 301,
					AdditionalAttributes: []attribute.KeyValue{
						attribute.String("key", "value"),
					},
				},
				MetricData: MetricData{
					RequestSize: 100,
					ElapsedTime: 300,
				},
			})

			tt.expectedFunc(server, t)
		})
	}
}

func TestNewMethod(t *testing.T) {
	testCases := []struct {
		method   string
		n        int
		want     attribute.KeyValue
		wantOrig attribute.KeyValue
	}{
		{
			method: http.MethodPost,
			n:      1,
			want:   attribute.String("http.request.method", "POST"),
		},
		{
			method:   "Put",
			n:        2,
			want:     attribute.String("http.request.method", "PUT"),
			wantOrig: attribute.String("http.request.method_original", "Put"),
		},
		{
			method:   "Unknown",
			n:        2,
			want:     attribute.String("http.request.method", "GET"),
			wantOrig: attribute.String("http.request.method_original", "Unknown"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			got, gotOrig := CurrentHTTPServer{}.method(tt.method)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOrig, gotOrig)
		})
	}
}

func TestHTTPClientStatus(t *testing.T) {
	tests := []struct {
		code int
		stat codes.Code
		msg  bool
	}{
		{0, codes.Error, true},
		{http.StatusContinue, codes.Unset, false},
		{http.StatusSwitchingProtocols, codes.Unset, false},
		{http.StatusProcessing, codes.Unset, false},
		{http.StatusEarlyHints, codes.Unset, false},
		{http.StatusOK, codes.Unset, false},
		{http.StatusCreated, codes.Unset, false},
		{http.StatusAccepted, codes.Unset, false},
		{http.StatusNonAuthoritativeInfo, codes.Unset, false},
		{http.StatusNoContent, codes.Unset, false},
		{http.StatusResetContent, codes.Unset, false},
		{http.StatusPartialContent, codes.Unset, false},
		{http.StatusMultiStatus, codes.Unset, false},
		{http.StatusAlreadyReported, codes.Unset, false},
		{http.StatusIMUsed, codes.Unset, false},
		{http.StatusMultipleChoices, codes.Unset, false},
		{http.StatusMovedPermanently, codes.Unset, false},
		{http.StatusFound, codes.Unset, false},
		{http.StatusSeeOther, codes.Unset, false},
		{http.StatusNotModified, codes.Unset, false},
		{http.StatusUseProxy, codes.Unset, false},
		{306, codes.Unset, false},
		{http.StatusTemporaryRedirect, codes.Unset, false},
		{http.StatusPermanentRedirect, codes.Unset, false},
		{http.StatusBadRequest, codes.Error, false},
		{http.StatusUnauthorized, codes.Error, false},
		{http.StatusPaymentRequired, codes.Error, false},
		{http.StatusForbidden, codes.Error, false},
		{http.StatusNotFound, codes.Error, false},
		{http.StatusMethodNotAllowed, codes.Error, false},
		{http.StatusNotAcceptable, codes.Error, false},
		{http.StatusProxyAuthRequired, codes.Error, false},
		{http.StatusRequestTimeout, codes.Error, false},
		{http.StatusConflict, codes.Error, false},
		{http.StatusGone, codes.Error, false},
		{http.StatusLengthRequired, codes.Error, false},
		{http.StatusPreconditionFailed, codes.Error, false},
		{http.StatusRequestEntityTooLarge, codes.Error, false},
		{http.StatusRequestURITooLong, codes.Error, false},
		{http.StatusUnsupportedMediaType, codes.Error, false},
		{http.StatusRequestedRangeNotSatisfiable, codes.Error, false},
		{http.StatusExpectationFailed, codes.Error, false},
		{http.StatusTeapot, codes.Error, false},
		{http.StatusMisdirectedRequest, codes.Error, false},
		{http.StatusUnprocessableEntity, codes.Error, false},
		{http.StatusLocked, codes.Error, false},
		{http.StatusFailedDependency, codes.Error, false},
		{http.StatusTooEarly, codes.Error, false},
		{http.StatusUpgradeRequired, codes.Error, false},
		{http.StatusPreconditionRequired, codes.Error, false},
		{http.StatusTooManyRequests, codes.Error, false},
		{http.StatusRequestHeaderFieldsTooLarge, codes.Error, false},
		{http.StatusUnavailableForLegalReasons, codes.Error, false},
		{499, codes.Error, false},
		{http.StatusInternalServerError, codes.Error, false},
		{http.StatusNotImplemented, codes.Error, false},
		{http.StatusBadGateway, codes.Error, false},
		{http.StatusServiceUnavailable, codes.Error, false},
		{http.StatusGatewayTimeout, codes.Error, false},
		{http.StatusHTTPVersionNotSupported, codes.Error, false},
		{http.StatusVariantAlsoNegotiates, codes.Error, false},
		{http.StatusInsufficientStorage, codes.Error, false},
		{http.StatusLoopDetected, codes.Error, false},
		{http.StatusNotExtended, codes.Error, false},
		{http.StatusNetworkAuthenticationRequired, codes.Error, false},
		{600, codes.Error, true},
	}

	for _, test := range tests {
		t.Run(strconv.Itoa(test.code), func(t *testing.T) {
			c, msg := HTTPClient{}.Status(test.code)
			assert.Equal(t, test.stat, c)
			if test.msg && msg == "" {
				t.Errorf("expected non-empty message for %d", test.code)
			} else if !test.msg && msg != "" {
				t.Errorf("expected empty message for %d, got: %s", test.code, msg)
			}
		})
	}
}
