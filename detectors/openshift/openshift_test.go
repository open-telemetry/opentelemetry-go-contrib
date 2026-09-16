// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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
	url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

// A rejected token or missing RBAC is a misconfigured detector, not evidence
// that the process does not run on OpenShift.
func TestDetectUnauthorized(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})

		res, err := newTestDetector(url).Detect(t.Context())
		require.Error(t, err)
		assert.NotErrorIs(t, err, resource.ErrPartialResource)
		assert.Nil(t, res)
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
	assert.ErrorContains(t, err, "aws.region")

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

// An empty token would produce an invalid Authorization header, so no request
// is sent.
func TestDetectEmptyServiceAccountToken(t *testing.T) {
	var requests int
	url := newFakeServerFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(awsInfra())
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(" \n"), 0o600))

	d := NewResourceDetector(WithAddress(url))
	d.tokenPath = tokenPath

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
	assert.Equal(t, 0, requests)
}

func newTLSFakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(awsInfra())
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDetectWithTLSConfig(t *testing.T) {
	srv := newTLSFakeServer(t)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())

	d := newTestDetector(srv.URL, WithTLSConfig(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	val, ok := res.Set().Value(semconv.K8SClusterNameKey)
	require.True(t, ok)
	assert.Equal(t, "test-d-bm4rt", val.AsString())
}

// Without WithTLSConfig the projected certificate authority is the root of
// trust.
func TestDetectReadsCertificateAuthority(t *testing.T) {
	srv := newTLSFakeServer(t)

	caPath := filepath.Join(t.TempDir(), "ca.crt")
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caPath, ca, 0o600))

	d := newTestDetector(srv.URL)
	d.caPath = caPath

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	val, ok := res.Set().Value(semconv.K8SClusterNameKey)
	require.True(t, ok)
	assert.Equal(t, "test-d-bm4rt", val.AsString())
}

func TestDetectInvalidCertificateAuthority(t *testing.T) {
	srv := newTLSFakeServer(t)

	caPath := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caPath, []byte("not a certificate"), 0o600))

	d := newTestDetector(srv.URL)
	d.caPath = caPath

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
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

// Kubernetes sets the host to a bare IPv6 address on IPv6 clusters. A URL
// authority needs it bracketed.
func TestAddressFromEnvIPv6(t *testing.T) {
	t.Setenv(hostEnvVar, "fd00:10:96::1")
	t.Setenv(portEnvVar, "443")

	address, ok := NewResourceDetector().address()
	require.True(t, ok)
	assert.Equal(t, "https://[fd00:10:96::1]:443", address)
}

// The request paths appended to the address already start with a separator, so
// a trailing slash on the configured address must not double it.
func TestDetectTrimsTrailingSlashFromAddress(t *testing.T) {
	var gotPath string
	url := newFakeServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(awsInfra())
	})

	_, err := newTestDetector(url + "/").Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, infrastructurePath, gotPath)
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

// TestComposition_MergeWithDefault guards against schema URL drift between
// this detector and the SDK. [resource.Merge] reports
// [resource.ErrSchemaURLConflict] and drops the schema URL when the two
// disagree, so this fails as soon as the semconv version here and the one
// behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	detected, err := newTestDetector(newFakeServer(t, awsInfra())).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// built-in detectors of go.opentelemetry.io/otel/sdk.
func TestComposition_WithCoreDetectors(t *testing.T) {
	d := newTestDetector(newFakeServer(t, awsInfra()))

	res, err := resource.New(t.Context(),
		resource.WithDetectors(d),
		resource.WithHost(),
		resource.WithFromEnv(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())

	val, ok := res.Set().Value(semconv.K8SClusterNameKey)
	require.True(t, ok)
	assert.Equal(t, "test-d-bm4rt", val.AsString())
}
