// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
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
func (detectorUtils *MockDetectorUtils) getConfigMap(ctx context.Context, namespace, name string) (map[string]string, error) {
	args := detectorUtils.Called(ctx, namespace, name)
	var cm map[string]string
	if v := args.Get(0); v != nil {
		cm = v.(map[string]string)
	}
	return cm, args.Error(1)
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
	detectorUtils.On("getConfigMap", mock.Anything, authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
	detectorUtils.On("getConfigMap", mock.Anything, cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
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

func TestConfigMapContextKeepsExistingDeadline(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	deadline := time.Now().Add(time.Hour)
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", mock.MatchedBy(func(ctx context.Context) bool {
		got, ok := ctx.Deadline()
		return ok && got.Equal(deadline)
	}), authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)

	isEks, err := isEKS(ctx, detectorUtils)
	require.NoError(t, err)
	assert.True(t, isEks)
	detectorUtils.AssertExpectations(t)
}

func TestConfigMapContextAddsDefaultTimeout(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	start := time.Now()

	detectorUtils.On("getConfigMap", mock.MatchedBy(func(ctx context.Context) bool {
		deadline, ok := ctx.Deadline()
		return ok && deadline.After(start) &&
			deadline.Sub(start) > defaultK8sAPICallTimeout-time.Second &&
			deadline.Sub(start) <= defaultK8sAPICallTimeout+time.Second
	}), cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)

	clusterName, err := getClusterName(t.Context(), detectorUtils)
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", clusterName)
	detectorUtils.AssertExpectations(t)
}

func TestConfigMapContextPreservesCancellation(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("getConfigMap", mock.MatchedBy(func(ctx context.Context) bool {
		return errors.Is(ctx.Err(), context.Canceled)
	}), authConfigmapNS, authConfigmapName).Return(nil, context.Canceled)

	isEks, err := isEKS(ctx, detectorUtils)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, isEks)
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
