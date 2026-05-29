// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const testInstanceJSON = `{
	"crn": "crn:v1:bluemix:public:is:eu-es-2:a/bab397cffebd40329900ec4b9039793a::instance:02x7_8340ad93-8b46-41cf-95b5-6585f54dd419",
	"id": "02x7_8340ad93-8b46-41cf-95b5-6585f54dd419",
	"image": {
		"id": "r050-8844669d-ac98-4de5-8651-109645ead299",
		"name": "ibm-ubuntu-24-04-4-minimal-amd64-1"
	},
	"name": "otel-collector",
	"profile": {"name": "nxf-1x1"},
	"zone": {"name": "eu-es-2"}
}`

func TestNewResourceDetector(t *testing.T) {
	var _ resource.Detector = NewResourceDetector()
	assert.NotNil(t, NewResourceDetector())
}

func TestNewResourceDetectorWithHTTPSProtocol(t *testing.T) {
	detector := NewResourceDetector(WithProtocol("https"))
	assert.Equal(t, "https://"+metadataHost, detector.endpoint)
}

func TestDetect(t *testing.T) {
	var tokenRequests atomic.Int32
	srv := newMetadataServer(t, &tokenRequests, http.StatusOK, testInstanceJSON)
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String(cloudPlatformIBMCloudVPC),
		semconv.CloudRegion("eu-es"),
		semconv.CloudAvailabilityZone("eu-es-2"),
		semconv.CloudAccountID("bab397cffebd40329900ec4b9039793a"),
		semconv.CloudResourceID("crn:v1:bluemix:public:is:eu-es-2:a/bab397cffebd40329900ec4b9039793a::instance:02x7_8340ad93-8b46-41cf-95b5-6585f54dd419"),
		semconv.HostID("02x7_8340ad93-8b46-41cf-95b5-6585f54dd419"),
		semconv.HostImageID("r050-8844669d-ac98-4de5-8651-109645ead299"),
		semconv.HostImageName("ibm-ubuntu-24-04-4-minimal-amd64-1"),
		semconv.HostName("otel-collector"),
		semconv.HostType("nxf-1x1"),
	)
	assert.Equal(t, expected, res)

	_, err = detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(1), tokenRequests.Load(), "second detection should reuse the metadata token")
}

func TestDetectReusesShortLivedToken(t *testing.T) {
	var tokenRequests atomic.Int32
	srv := newMetadataServerWithTokenBody(t, &tokenRequests, http.StatusOK, testInstanceJSON, `{"access_token":"test-token","expires_in":1}`)
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	_, err := detector.Detect(t.Context())
	require.NoError(t, err)

	_, err = detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(1), tokenRequests.Load(), "second detection should reuse a still-valid short-lived token")
}

func TestDetectRefreshesExpiredToken(t *testing.T) {
	var tokenRequests atomic.Int32
	srv := newMetadataServerWithTokenBody(t, &tokenRequests, http.StatusOK, testInstanceJSON, `{"access_token":"test-token","expires_in":-1}`)
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	_, err := detector.Detect(t.Context())
	require.NoError(t, err)

	_, err = detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(2), tokenRequests.Load(), "second detection should refresh an expired metadata token")
}

func TestDetectMetadataUnavailable(t *testing.T) {
	srv := newMetadataServer(t, nil, http.StatusServiceUnavailable, "service unavailable")
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectInvalidMetadataJSON(t *testing.T) {
	srv := newMetadataServer(t, nil, http.StatusOK, "{")
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectTokenUnavailable(t *testing.T) {
	srv := newMetadataServerWithTokenResponse(t, nil, http.StatusInternalServerError, "token unavailable", http.StatusOK, testInstanceJSON)
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectInvalidTokenJSON(t *testing.T) {
	srv := newMetadataServerWithTokenBody(t, nil, http.StatusOK, testInstanceJSON, "{")
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectMissingAccessToken(t *testing.T) {
	srv := newMetadataServerWithTokenBody(t, nil, http.StatusOK, testInstanceJSON, `{"expires_in":300}`)
	defer srv.Close()

	detector := NewResourceDetector()
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestInstanceMetadataCreateRequestError(t *testing.T) {
	detector := NewResourceDetector()
	detector.endpoint = "://bad"
	detector.token = "test-token"
	detector.tokenExpiry = time.Now().Add(time.Minute)

	meta, err := detector.instanceMetadata(t.Context())
	require.Error(t, err)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "failed to create metadata request")
}

func TestInstanceMetadataRequestError(t *testing.T) {
	detector := NewResourceDetector()
	detector.endpoint = "http://metadata.test"
	detector.client = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("metadata request failed")
		}),
	}
	detector.token = "test-token"
	detector.tokenExpiry = time.Now().Add(time.Minute)

	meta, err := detector.instanceMetadata(t.Context())
	require.Error(t, err)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "failed to get instance metadata")
}

func TestGetTokenCreateRequestError(t *testing.T) {
	detector := NewResourceDetector()
	detector.endpoint = "://bad"

	token, err := detector.getToken(t.Context())
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "failed to create token request")
}

func TestGetTokenRequestError(t *testing.T) {
	detector := NewResourceDetector()
	detector.endpoint = "http://metadata.test"
	detector.client = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("token request failed")
		}),
	}

	token, err := detector.getToken(t.Context())
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "failed to get metadata token")
}

func TestDetectInvalidProtocol(t *testing.T) {
	res, err := NewResourceDetector(WithProtocol("ftp")).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), `invalid protocol "ftp"`)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	srv := newMetadataServer(t, nil, http.StatusOK, testInstanceJSON)
	defer srv.Close()

	detector := NewResourceDetector(WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.HostImageNameKey)))
	detector.endpoint = srv.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostImageNameKey)
	assert.False(t, ok)
	_, ok = res.Set().Value(semconv.HostImageIDKey)
	assert.True(t, ok)
}

func TestRegionFromZone(t *testing.T) {
	assert.Equal(t, "us-south", regionFromZone("us-south-1"))
	assert.Equal(t, "eu-de", regionFromZone("eu-de-2"))
	assert.Equal(t, "nodash", regionFromZone("nodash"))
	assert.Empty(t, regionFromZone(""))
}

func TestAccountIDFromCRN(t *testing.T) {
	assert.Equal(t, "123456789012", accountIDFromCRN("crn:v1:bluemix:public:is:us-south-1:a/123456789012::instance:0717_xxx"))
	assert.Equal(t, "123456789012", accountIDFromCRN("crn:v1:bluemix:public:is:us-south-1:123456789012::instance:0717_xxx"))
	assert.Empty(t, accountIDFromCRN("crn:v1:bluemix"))
	assert.Empty(t, accountIDFromCRN(""))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMetadataServer(t *testing.T, tokenRequests *atomic.Int32, metadataStatus int, metadataBody string) *httptest.Server {
	t.Helper()

	return newMetadataServerWithTokenBody(t, tokenRequests, metadataStatus, metadataBody, `{"access_token":"test-token","expires_in":300}`)
}

func newMetadataServerWithTokenBody(t *testing.T, tokenRequests *atomic.Int32, metadataStatus int, metadataBody, tokenBody string) *httptest.Server {
	t.Helper()

	return newMetadataServerWithTokenResponse(t, tokenRequests, http.StatusOK, tokenBody, metadataStatus, metadataBody)
}

func newMetadataServerWithTokenResponse(t *testing.T, tokenRequests *atomic.Int32, tokenStatus int, tokenBody string, metadataStatus int, metadataBody string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == tokenPath && r.Method == http.MethodPut:
			if r.Header.Get(metadataFlavorKey) != metadataFlavorValue {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if tokenRequests != nil {
				tokenRequests.Add(1)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tokenStatus)
			_, _ = w.Write([]byte(tokenBody))
		case r.URL.Path == instancePath && r.Method == http.MethodGet:
			if r.Header.Get("Authorization") != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(metadataStatus)
			_, _ = w.Write([]byte(metadataBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
