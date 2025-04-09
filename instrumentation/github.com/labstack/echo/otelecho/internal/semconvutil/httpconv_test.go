// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/semconvutil/httpconv_test.go.tmpl

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconvutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func TestHTTPClientResponse(t *testing.T) {
	const stat, n = 201, 397
	resp := &http.Response{
		StatusCode:    stat,
		ContentLength: n,
	}
	got := HTTPClientResponse(resp, nil)
	assert.Equal(t, 2, cap(got), "slice capacity")
	assert.ElementsMatch(t, []attribute.KeyValue{
		attribute.Key("http.status_code").Int(stat),
		attribute.Key("http.response_content_length").Int(n),
	}, got)
}

func TestHTTPSClientRequest(t *testing.T) {
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "127.0.0.1:443",
			Path:   "/resource",
		},
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}

	assert.ElementsMatch(
		t,
		[]attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.url", "https://127.0.0.1:443/resource"),
			attribute.String("net.peer.name", "127.0.0.1"),
		},
		HTTPClientRequest(req, nil),
	)
}

func TestHTTPSClientRequestMetrics(t *testing.T) {
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "127.0.0.1:443",
			Path:   "/resource",
		},
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}

	assert.ElementsMatch(
		t,
		[]attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("net.peer.name", "127.0.0.1"),
		},
		HTTPClientRequestMetrics(req),
	)
}

func TestHTTPClientRequest(t *testing.T) {
	const (
		user  = "alice"
		n     = 128
		agent = "Go-http-client/1.1"
	)
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:8080",
			Path:   "/resource",
		},
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header: http.Header{
			"User-Agent": []string{agent},
		},
		ContentLength: n,
	}
	req.SetBasicAuth(user, "pswrd")

	assert.ElementsMatch(
		t,
		[]attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.url", "http://127.0.0.1:8080/resource"),
			attribute.String("net.peer.name", "127.0.0.1"),
			attribute.Int("net.peer.port", 8080),
			attribute.String("user_agent.original", agent),
			attribute.Int("http.request_content_length", n),
		},
		HTTPClientRequest(req, nil),
	)
}

func TestHTTPClientRequestMetrics(t *testing.T) {
	const (
		user  = "alice"
		n     = 128
		agent = "Go-http-client/1.1"
	)
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:8080",
			Path:   "/resource",
		},
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header: http.Header{
			"User-Agent": []string{agent},
		},
		ContentLength: n,
	}
	req.SetBasicAuth(user, "pswrd")

	assert.ElementsMatch(
		t,
		[]attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("net.peer.name", "127.0.0.1"),
			attribute.Int("net.peer.port", 8080),
		},
		HTTPClientRequestMetrics(req),
	)
}

func TestHTTPClientRequestRequired(t *testing.T) {
	req := new(http.Request)
	var got []attribute.KeyValue
	assert.NotPanics(t, func() { got = HTTPClientRequest(req, nil) })
	want := []attribute.KeyValue{
		attribute.String("http.method", "GET"),
		attribute.String("http.url", ""),
		attribute.String("net.peer.name", ""),
	}
	assert.Equal(t, want, got)
}

func TestHTTPServerRequest(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		requestModifierFn     func(r *http.Request)
		httpServerRequestOpts HTTPServerRequestOptions

		wantClientIP string
	}{
		{
			name:         "with a client IP from the network",
			wantClientIP: "1.2.3.4",
		},
		{
			name: "with a client IP from x-forwarded-for header",
			requestModifierFn: func(r *http.Request) {
				r.Header.Add("X-Forwarded-For", "5.6.7.8")
			},
			wantClientIP: "5.6.7.8",
		},
		{
			name: "with a client IP in options",
			requestModifierFn: func(r *http.Request) {
				r.Header.Add("X-Forwarded-For", "5.6.7.8")
			},
			httpServerRequestOpts: HTTPServerRequestOptions{
				HTTPClientIP: "9.8.7.6",
			},
			wantClientIP: "9.8.7.6",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reqCh := make(chan *http.Request, 1)
			handler := func(w http.ResponseWriter, r *http.Request) {
				r.RemoteAddr = "1.2.3.4:5678"
				reqCh <- r
				w.WriteHeader(http.StatusOK)
			}

			srv := httptest.NewServer(http.HandlerFunc(handler))
			defer srv.Close()

			srvURL, err := url.Parse(srv.URL)
			require.NoError(t, err)
			srvPort, err := strconv.ParseInt(srvURL.Port(), 10, 32)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			if tt.requestModifierFn != nil {
				tt.requestModifierFn(req)
			}

			resp, err := srv.Client().Do(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			var got *http.Request
			select {
			case got = <-reqCh:
				// All good
			case <-time.After(5 * time.Second):
				t.Fatal("Did not receive a signal in 5s")
			}

			peer, peerPort := splitHostPort(got.RemoteAddr)

			const user = "alice"
			got.SetBasicAuth(user, "pswrd")

			assert.ElementsMatch(t,
				[]attribute.KeyValue{
					attribute.String("http.method", "GET"),
					attribute.String("http.scheme", "http"),
					attribute.String("net.host.name", srvURL.Hostname()),
					attribute.Int("net.host.port", int(srvPort)),
					attribute.String("net.sock.peer.addr", peer),
					attribute.Int("net.sock.peer.port", peerPort),
					attribute.String("user_agent.original", "Go-http-client/1.1"),
					attribute.String("http.client_ip", tt.wantClientIP),
					attribute.String("net.protocol.version", "1.1"),
					attribute.String("http.target", "/"),
				},
				HTTPServerRequest("", got, tt.httpServerRequestOpts, nil))
		})
	}
}

func TestHTTPServerRequestMetrics(t *testing.T) {
	got := make(chan *http.Request, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		got <- r
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)
	srvPort, err := strconv.ParseInt(srvURL.Port(), 10, 32)
	require.NoError(t, err)

	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	var req *http.Request
	select {
	case req = <-got:
		// All good
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive a signal in 5s")
	}

	assert.ElementsMatch(t,
		[]attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.scheme", "http"),
			attribute.String("net.host.name", srvURL.Hostname()),
			attribute.Int("net.host.port", int(srvPort)),
			attribute.String("net.protocol.name", "http"),
			attribute.String("net.protocol.version", "1.1"),
		},
		HTTPServerRequestMetrics("", req))
}

func TestHTTPServerName(t *testing.T) {
	req := new(http.Request)
	var got []attribute.KeyValue
	const (
		host = "test.semconv.server"
		port = 8080
	)
	portStr := strconv.Itoa(port)
	server := host + ":" + portStr
	assert.NotPanics(t, func() { got = HTTPServerRequest(server, req, HTTPServerRequestOptions{}, nil) })
	assert.Contains(t, got, attribute.String("net.host.name", host))
	assert.Contains(t, got, attribute.Int("net.host.port", port))

	req = &http.Request{Host: "alt.host.name:" + portStr}
	// The server parameter does not include a port, ServerRequest should use
	// the port in the request Host field.
	assert.NotPanics(t, func() { got = HTTPServerRequest(host, req, HTTPServerRequestOptions{}, nil) })
	assert.Contains(t, got, attribute.String("net.host.name", host))
	assert.Contains(t, got, attribute.Int("net.host.port", port))
}

func TestHTTPServerRequestFailsGracefully(t *testing.T) {
	req := new(http.Request)
	var got []attribute.KeyValue
	assert.NotPanics(t, func() { got = HTTPServerRequest("", req, HTTPServerRequestOptions{}, nil) })
	want := []attribute.KeyValue{
		attribute.String("http.method", "GET"),
		attribute.String("http.scheme", "http"),
		attribute.String("net.host.name", ""),
	}
	assert.ElementsMatch(t, want, got)
}

func TestHTTPMethod(t *testing.T) {
	assert.Equal(t, attribute.String("http.method", "POST"), hc.method("POST"))
	assert.Equal(t, attribute.String("http.method", "GET"), hc.method(""))
	assert.Equal(t, attribute.String("http.method", "garbage"), hc.method("garbage"))
}

func TestHTTPScheme(t *testing.T) {
	assert.Equal(t, attribute.String("http.scheme", "http"), hc.scheme(false))
	assert.Equal(t, attribute.String("http.scheme", "https"), hc.scheme(true))
}

func TestHTTPServerClientIP(t *testing.T) {
	tests := []struct {
		xForwardedFor string
		want          string
	}{
		{"", ""},
		{"127.0.0.1", "127.0.0.1"},
		{"127.0.0.1,127.0.0.5", "127.0.0.1"},
	}
	for _, test := range tests {
		got := serverClientIP(test.xForwardedFor)
		assert.Equal(t, test.want, got, test.xForwardedFor)
	}
}

func TestRequiredHTTPPort(t *testing.T) {
	tests := []struct {
		https bool
		port  int
		want  int
	}{
		{true, 443, -1},
		{true, 80, 80},
		{true, 8081, 8081},
		{false, 443, 443},
		{false, 80, -1},
		{false, 8080, 8080},
	}
	for _, test := range tests {
		got := requiredHTTPPort(test.https, test.port)
		assert.Equalf(t, test.want, got, "HTTPS: %t, Port: %d", test.https, test.port)
	}
}

func TestFirstHostPort(t *testing.T) {
	host, port := "127.0.0.1", 8080
	hostport := "127.0.0.1:8080"
	sources := [][]string{
		{hostport},
		{"", hostport},
		{"", "", hostport},
		{"", "", hostport, ""},
		{"", "", hostport, "127.0.0.3:80"},
	}

	for _, src := range sources {
		h, p := firstHostPort(src...)
		assert.Equal(t, host, h, "%+v", src)
		assert.Equal(t, port, p, "%+v", src)
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
			c, msg := HTTPClientStatus(test.code)
			assert.Equal(t, test.stat, c)
			if test.msg && msg == "" {
				t.Errorf("expected non-empty message for %d", test.code)
			} else if !test.msg && msg != "" {
				t.Errorf("expected empty message for %d, got: %s", test.code, msg)
			}
		})
	}
}

func TestHTTPServerStatus(t *testing.T) {
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
		{http.StatusBadRequest, codes.Unset, false},
		{http.StatusUnauthorized, codes.Unset, false},
		{http.StatusPaymentRequired, codes.Unset, false},
		{http.StatusForbidden, codes.Unset, false},
		{http.StatusNotFound, codes.Unset, false},
		{http.StatusMethodNotAllowed, codes.Unset, false},
		{http.StatusNotAcceptable, codes.Unset, false},
		{http.StatusProxyAuthRequired, codes.Unset, false},
		{http.StatusRequestTimeout, codes.Unset, false},
		{http.StatusConflict, codes.Unset, false},
		{http.StatusGone, codes.Unset, false},
		{http.StatusLengthRequired, codes.Unset, false},
		{http.StatusPreconditionFailed, codes.Unset, false},
		{http.StatusRequestEntityTooLarge, codes.Unset, false},
		{http.StatusRequestURITooLong, codes.Unset, false},
		{http.StatusUnsupportedMediaType, codes.Unset, false},
		{http.StatusRequestedRangeNotSatisfiable, codes.Unset, false},
		{http.StatusExpectationFailed, codes.Unset, false},
		{http.StatusTeapot, codes.Unset, false},
		{http.StatusMisdirectedRequest, codes.Unset, false},
		{http.StatusUnprocessableEntity, codes.Unset, false},
		{http.StatusLocked, codes.Unset, false},
		{http.StatusFailedDependency, codes.Unset, false},
		{http.StatusTooEarly, codes.Unset, false},
		{http.StatusUpgradeRequired, codes.Unset, false},
		{http.StatusPreconditionRequired, codes.Unset, false},
		{http.StatusTooManyRequests, codes.Unset, false},
		{http.StatusRequestHeaderFieldsTooLarge, codes.Unset, false},
		{http.StatusUnavailableForLegalReasons, codes.Unset, false},
		{499, codes.Unset, false},
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
		c, msg := HTTPServerStatus(test.code)
		assert.Equal(t, test.stat, c)
		if test.msg && msg == "" {
			t.Errorf("expected non-empty message for %d", test.code)
		} else if !test.msg && msg != "" {
			t.Errorf("expected empty message for %d, got: %s", test.code, msg)
		}
	}
}
