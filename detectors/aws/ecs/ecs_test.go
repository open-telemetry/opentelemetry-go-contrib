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

package aws

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

type private struct {
}

func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func TestDetect(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("Hostname").Return("container-Name")
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	labels := []label.KeyValue{
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
	}
	expectedResource := resource.New(labels...)

	//Call ECS Resource detector to detect resources
	ecsDetector := ECS{}
	resource, _ := ecsDetector.Detect(context.TODO())

	assert.Equal(t, resource.Attributes(), expectedResource.Attributes(), "Resource returned is incorrect")
}

func TestNotOnEcs(t *testing.T) {
	os.Clearenv()
	ecs := ECS{}
	resource, err := ecs.Detect(context.TODO())

	assert.Equal(t, errNotOnECS, err)
	assert.Equal(t, 0, len(resource.Attributes()))
}
