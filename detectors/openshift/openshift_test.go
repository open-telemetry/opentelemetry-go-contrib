// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const testToken = "test-token"

// newFakeServer starts an httptest server serving infra as JSON on the
// Infrastructure status path and returns its URL. The server is closed via
// t.Cleanup.
func newFakeServer(t *testing.T, infra infrastructureResponse) string {
	t.Helper()
	return newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(infra)
	})
}

func newFakeServerFunc(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at url instead of the real
// OpenShift API server. Its credential paths point at a directory that does
// not exist so that a test never reads the credentials of the host it runs on.
func newTestDetector(url string, opts ...Option) *ResourceDetector {
	opts = append([]Option{WithAddress(url), WithToken(testToken)}, opts...)
	d := NewResourceDetector(opts...)
	d.tokenPath = filepath.Join("testdata", "does-not-exist", "token")
	d.caPath = filepath.Join("testdata", "does-not-exist", "ca.crt")
	return d
}

func awsInfra() infrastructureResponse {
	return infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-d-bm4rt",
			PlatformStatus: platformStatus{
				Type: "AWS",
				AWS:  awsPlatform{Region: "us-east-1"},
			},
		},
	}
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultTokenPath, d.tokenPath)
	assert.Equal(t, defaultCAPath, d.caPath)
}

func TestDetectAWS(t *testing.T) {
	url := newFakeServer(t, awsInfra())

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-d-bm4rt"),
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSOpenShift,
		semconv.CloudRegion("us-east-1"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectAzure(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-azure",
			PlatformStatus: platformStatus{
				Type:  "Azure",
				Azure: azurePlatform{CloudName: "AzurePublicCloud"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-azure"),
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureOpenShift,
		semconv.CloudRegion("azurepubliccloud"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectGCP(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-gcp",
			PlatformStatus: platformStatus{
				Type: "GCP",
				GCP:  gcpPlatform{Region: "us-central1"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-gcp"),
		semconv.CloudProviderGCP,
		semconv.CloudPlatformGCPOpenShift,
		semconv.CloudRegion("us-central1"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectIBMCloud(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-ibm",
			PlatformStatus: platformStatus{
				Type:     "IBMCloud",
				IBMCloud: ibmCloudPlatform{Location: "eu-de"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-ibm"),
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformIBMCloudOpenShift,
		semconv.CloudRegion("eu-de"),
	)
	assert.Equal(t, expected, res)
}

// OpenStack clusters report only the region: semantic conventions define no
// cloud.provider value for OpenStack and no cloud.platform value for OpenShift
// on OpenStack.
func TestDetectOpenStack(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-osp",
			PlatformStatus: platformStatus{
				Type:      "OpenStack",
				OpenStack: openStackPlatform{CloudName: "MyCloud"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-osp"),
		semconv.CloudRegion("mycloud"),
	)
	assert.Equal(t, expected, res)
}

// A cluster that does not run on a cloud provider reports no cloud attributes
// and is not a partial resource.
func TestDetectBareMetal(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-bm",
			PlatformStatus:     platformStatus{Type: "BareMetal"},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-bm"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectRegionIsNormalized(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-upper",
			PlatformStatus: platformStatus{
				Type: "AWS",
				AWS:  awsPlatform{Region: "US-EAST-1"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.CloudRegionKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("us-east-1"), val)
}

func TestDetectSendsBearerToken(t *testing.T) {
	var (
		gotAuth string
		gotPath string
	)
	url := newFakeServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(awsInfra())
	})

	_, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Bearer "+testToken, gotAuth)
	assert.Equal(t, infrastructurePath, gotPath)
}

// The Infrastructure document is fetched exactly once per detection.
func TestDetectFetchesOnce(t *testing.T) {
	var requests int
	url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(awsInfra())
	})

	_, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, requests)
}

// A plain Kubernetes API server does not serve the OpenShift config API group
// and answers 404.
func TestDetectNotOpenShift(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusUnauthorized} {
		url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})

		res, err := newTestDetector(url).Detect(t.Context())
		require.NoError(t, err)
		assert.Equal(t, resource.Empty(), res)
	}
}

func TestDetectAPIServerUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectServerError(t *testing.T) {
	url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectMalformedBody(t *testing.T) {
	url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

// A cancelled context must not be reported as "not running on OpenShift".
func TestDetectContextCancelled(t *testing.T) {
	url := newFakeServer(t, awsInfra())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := newTestDetector(url).Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}

func TestDetectPartialMissingClusterName(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			PlatformStatus: platformStatus{
				Type: "AWS",
				AWS:  awsPlatform{Region: "us-east-1"},
			},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSOpenShift,
		semconv.CloudRegion("us-east-1"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectPartialMissingRegion(t *testing.T) {
	url := newFakeServer(t, infrastructureResponse{
		Status: infrastructureStatus{
			InfrastructureName: "test-no-region",
			PlatformStatus:     platformStatus{Type: "AWS"},
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-no-region"),
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSOpenShift,
	)
	assert.Equal(t, expected, res)
}

func TestDetectNotInCluster(t *testing.T) {
	t.Setenv(hostEnvVar, "")
	t.Setenv(portEnvVar, "")

	d := NewResourceDetector()
	d.tokenPath = filepath.Join("testdata", "does-not-exist", "token")
	d.caPath = filepath.Join("testdata", "does-not-exist", "ca.crt")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectMissingServiceAccountToken(t *testing.T) {
	t.Setenv(hostEnvVar, "10.0.0.1")
	t.Setenv(portEnvVar, "443")

	d := NewResourceDetector()
	d.tokenPath = filepath.Join("testdata", "does-not-exist", "token")
	d.caPath = filepath.Join("testdata", "does-not-exist", "ca.crt")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

// Without WithToken the projected service account token is used, with any
// trailing newline trimmed off.
func TestDetectReadsServiceAccountToken(t *testing.T) {
	var gotAuth string
	url := newFakeServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(awsInfra())
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("projected-token\n"), 0o600))

	d := NewResourceDetector(WithAddress(url))
	d.tokenPath = tokenPath

	_, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Bearer projected-token", gotAuth)
}

func TestDetectMissingCertificateAuthority(t *testing.T) {
	d := newTestDetector("https://10.0.0.1:443")

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestAddressFromEnv(t *testing.T) {
	t.Setenv(hostEnvVar, "10.0.0.1")
	t.Setenv(portEnvVar, "443")

	address, ok := NewResourceDetector().address()
	require.True(t, ok)
	assert.Equal(t, "https://10.0.0.1:443", address)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, awsInfra())

	filter := func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.K8SClusterNameKey
	}

	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName("test-d-bm4rt"),
	)
	assert.Equal(t, expected, res)
}
