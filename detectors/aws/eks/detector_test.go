// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"k8s.io/client-go/rest"
)

type MockDetectorUtils struct {
	mock.Mock
}

// Mock function for fileExists().
func (detectorUtils *MockDetectorUtils) fileExists(filename string) bool {
	args := detectorUtils.Called(filename)
	return args.Bool(0)
}

// Mock function for getConfigMap().
func (detectorUtils *MockDetectorUtils) getConfigMap(_ context.Context, namespace, name string) (map[string]string, error) {
	args := detectorUtils.Called(namespace, name)
	return args.Get(0).(map[string]string), args.Error(1)
}

// Mock function for getContainerID().
func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

// Tests EKS resource detector running in EKS environment.
func TestEks(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	// Mock functions and set expectations
	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	// Expected resource object
	eksResourceLabels := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName("my-cluster"),
		semconv.ContainerID("0123456789A"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, eksResourceLabels...)

	// Call EKS Resource detector to detect resources
	eksResourceDetector := resourceDetector{utils: detectorUtils}
	resourceObj, err := eksResourceDetector.Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, expectedResource, resourceObj, "Resource object returned is incorrect")
	detectorUtils.AssertExpectations(t)
}

// Tests EKS resource detector not running in EKS environment.
func TestNotEKS(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	k8sTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// Mock functions and set expectations
	detectorUtils.On("fileExists", k8sTokenPath).Return(false)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}

// Tests EKS resource detector not running K8S at all.
func TestNotK8S(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	detector := resourceDetector{utils: detectorUtils, err: rest.ErrNotInCluster}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}

func TestGetConfigMapSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/namespaces/kube-system/configmaps/aws-auth", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"data":{"mapRoles":"test-role"}}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	utils := &eksDetectorUtils{host: srv.URL, client: srv.Client()}
	data, err := utils.getConfigMap(t.Context(), authConfigmapNS, authConfigmapName)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"mapRoles": "test-role"}, data)
}

func TestGetConfigMapNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	utils := &eksDetectorUtils{host: srv.URL, client: srv.Client()}
	_, err := utils.getConfigMap(t.Context(), authConfigmapNS, authConfigmapName)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unexpected status")
}

// deadlineRoundTripper records the deadline of the request context and
// returns a static ConfigMap response without touching the network.
type deadlineRoundTripper struct {
	deadline time.Time
	ok       bool
}

func (rt *deadlineRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.deadline, rt.ok = req.Context().Deadline()
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		Request:    req,
	}, nil
}

func TestGetConfigMapRequestDeadline(t *testing.T) {
	t.Run("no caller deadline is bounded by configMapTimeout", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		before := time.Now()
		_, err := utils.getConfigMap(context.WithoutCancel(t.Context()), authConfigmapNS, authConfigmapName)
		after := time.Now()
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.WithinRange(t, rt.deadline, before.Add(configMapTimeout), after.Add(configMapTimeout))
	})

	t.Run("shorter caller deadline is preserved", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		callerDeadline := time.Now().Add(time.Second)
		ctx, cancel := context.WithDeadline(t.Context(), callerDeadline)
		t.Cleanup(cancel)

		_, err := utils.getConfigMap(ctx, authConfigmapNS, authConfigmapName)
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.Equal(t, callerDeadline, rt.deadline)
	})
}
