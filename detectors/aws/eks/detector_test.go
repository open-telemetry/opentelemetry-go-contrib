// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

// Tests that a client construction failure other than [rest.ErrNotInCluster] is
// reported as an error rather than silently skipped.
func TestConstructionError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	errConstruction := errors.New("failed to create config")

	detector := resourceDetector{utils: detectorUtils, err: errConstruction}
	r, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, errConstruction)
	assert.Nil(t, r, "Resource object should be nil")
	detectorUtils.AssertExpectations(t)
}

// Tests that a pod missing the service account certificate is not treated as K8s.
func TestMissingCertFile(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(false)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}

// Tests that a failure to read the aws-auth ConfigMap aborts detection.
func TestAuthConfigMapError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	errConfigMap := errors.New("forbidden")

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).
		Return(map[string]string(nil), errConfigMap)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, errConfigMap)
	assert.Nil(t, r, "Resource object should be nil")
	detectorUtils.AssertExpectations(t)
}

// Tests that a Kubernetes cluster without the aws-auth ConfigMap is not EKS.
func TestAuthConfigMapAbsent(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).
		Return(map[string]string(nil), nil)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}

// Tests that a failure to read the cluster-info ConfigMap aborts detection even
// though the environment was already confirmed to be EKS.
func TestClusterInfoConfigMapError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	errConfigMap := errors.New("not found")

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).
		Return(map[string]string(nil), errConfigMap)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, errConfigMap)
	assert.Nil(t, r, "Resource object should be nil")
	detectorUtils.AssertExpectations(t)
}

// Tests that a cluster-info ConfigMap without a cluster.name key omits the
// attribute instead of failing.
func TestMissingClusterName(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{}, nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	expectedResource := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.ContainerID("0123456789A"),
	)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedResource, r, "Resource object returned is incorrect")
	detectorUtils.AssertExpectations(t)
}

// Tests that a container ID lookup failure discards the attributes that were
// already collected.
func TestContainerIDError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	errContainerID := errors.New("cannot read containerID")

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
	detectorUtils.On("getContainerID").Return("", errContainerID)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, errContainerID)
	assert.Nil(t, r, "Resource object should be nil")
	detectorUtils.AssertExpectations(t)
}

// Tests that an empty container ID omits the attribute instead of failing.
func TestEmptyContainerID(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
	detectorUtils.On("getContainerID").Return("", nil)

	expectedResource := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName("my-cluster"),
	)

	detector := resourceDetector{utils: detectorUtils}
	r, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedResource, r, "Resource object returned is incorrect")
	detectorUtils.AssertExpectations(t)
}
