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

package ecs

import (
	"context"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

// Create interface for functions that need to be mocked
type MockDetectorUtils struct {
	mock.Mock
}

func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func (detectorUtils *MockDetectorUtils) getContainerName() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

//succesfully return resource when process is running on Amazon ECS environment
func TestDetect(t *testing.T) {
	os.Clearenv()
	os.Setenv(metadataV3EnvVar, "3")
	os.Setenv(metadataV4EnvVar, "4")

	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	labels := []label.KeyValue{
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
	}
	expectedResource := resource.NewWithAttributes(labels...)
	detector := ResourceDetector{detectorUtils}
	resource, _ := detector.Detect(context.Background())

	assert.Equal(t, resource, expectedResource, "Resource returned is incorrect")
}

//returns empty resource when detector cannot read container ID
func TestDetectCannotReadContainerID(t *testing.T) {
	os.Clearenv()
	os.Setenv(metadataV3EnvVar, "3")
	os.Setenv(metadataV4EnvVar, "4")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("", errCannotReadContainerID)

	detector := ResourceDetector{detectorUtils}
	resource, err := detector.Detect(context.Background())

	assert.Equal(t, errCannotReadContainerID, err)
	assert.Equal(t, 0, len(resource.Attributes()))
}

//returns empty resource when process is not running ECS
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := ResourceDetector{}
	resource, err := detector.Detect(context.Background())

	assert.Equal(t, errNotOnECS, err)
	assert.Equal(t, 0, len(resource.Attributes()))
}
