// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
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

func TestConfigMapContext(t *testing.T) {
	for _, operation := range []string{"isEKS", "getClusterName"} {
		for _, state := range []string{"no deadline", "deadline", "canceled", "expired"} {
			t.Run(operation+"/"+state, func(t *testing.T) {
				ctx := t.Context()
				switch state {
				case "deadline":
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, time.Hour)
					defer cancel()
				case "canceled":
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				case "expired":
					var cancel context.CancelFunc
					ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
					defer cancel()
				}

				utils := new(MockDetectorUtils)
				namespace, name := cwConfigmapNS, cwConfigmapName
				if operation == "isEKS" {
					utils.On("fileExists", k8sTokenPath).Return(true)
					utils.On("fileExists", k8sCertPath).Return(true)
					namespace, name = authConfigmapNS, authConfigmapName
				}
				utils.On("getConfigMap", mock.Anything, namespace, name).
					Run(func(args mock.Arguments) {
						// Both requests must receive the caller's context unchanged,
						// including when the caller has not set a deadline.
						assert.Same(t, ctx, args.Get(0))
					}).Return(map[string]string{"cluster.name": "my-cluster"}, ctx.Err()).Once()

				var err error
				if operation == "isEKS" {
					var detected bool
					detected, err = isEKS(ctx, utils)
					assert.Equal(t, ctx.Err() == nil, detected)
				} else {
					var name string
					name, err = getClusterName(ctx, utils)
					if ctx.Err() == nil {
						assert.Equal(t, "my-cluster", name)
					} else {
						assert.Empty(t, name)
					}
				}
				if ctx.Err() == nil {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, ctx.Err())
				}
				utils.AssertExpectations(t)
			})
		}
	}
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
