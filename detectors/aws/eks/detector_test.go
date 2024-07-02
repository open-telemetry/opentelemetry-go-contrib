// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
func (detectorUtils *MockDetectorUtils) getConfigMap(_ context.Context, namespace string, name string) (map[string]string, error) {
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
	resourceObj, err := eksResourceDetector.Detect(context.Background())
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
	r, err := detector.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}
