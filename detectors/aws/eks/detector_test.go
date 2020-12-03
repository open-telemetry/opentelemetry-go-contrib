// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eks

import (
	"context"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

type MockDetectorUtils struct {
	mock.Mock
}

// Mock function for fileExists()
func (detectorUtils *MockDetectorUtils) fileExists(filename string) bool {
	args := detectorUtils.Called(filename)
	return args.Bool(0)
}

// Mock function for fetchString()
func (detectorUtils *MockDetectorUtils) fetchString(httpMethod string, URL string) (string, error) {
	args := detectorUtils.Called(httpMethod, URL)
	return args.String(0), args.Error(1)
}

// Mock function for getContainerID()
func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

// Tests EKS resource detector running in EKS environment
func TestEks(t *testing.T) {

	detectorUtils := new(MockDetectorUtils)

	// Mock functions and set expectations
	detectorUtils.On("fileExists", k8sTokenPath).Return(true)
	detectorUtils.On("fileExists", k8sCertPath).Return(true)
	detectorUtils.On("fetchString", "GET", k8sSvcURL+authConfigmapPath).Return("not empty", nil)
	detectorUtils.On("fetchString", "GET", k8sSvcURL+cwConfigmapPath).Return(`{"data":{"cluster.name":"my-cluster"}}`, nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	// Expected resource object
	eksResourceLabels := []label.KeyValue{
		semconv.K8SClusterNameKey.String("my-cluster"),
		semconv.ContainerIDKey.String("0123456789A"),
	}
	expectedResource := resource.NewWithAttributes(eksResourceLabels...)

	// Call EKS Resource detector to detect resources
	eksResourceDetector := ResourceDetector{detectorUtils}
	resourceObj, err := eksResourceDetector.Detect(context.Background())
	require.NoError(t, err)

	assert.Equal(t, expectedResource, resourceObj, "Resource object returned is incorrect")
	detectorUtils.AssertExpectations(t)
}

// Tests EKS resource detector not running in EKS environment
func TestNotEKS(t *testing.T) {

	detectorUtils := new(MockDetectorUtils)

	k8sTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// Mock functions and set expectations
	detectorUtils.On("fileExists", k8sTokenPath).Return(false)

	detector := ResourceDetector{detectorUtils}
	r, err := detector.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), r, "Resource object should be empty")
	detectorUtils.AssertExpectations(t)
}
