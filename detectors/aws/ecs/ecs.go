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
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

const (
	TypeStr           = "ecs"
	metadataV3EnvVar  = "ECS_CONTAINER_METADATA_URI"
	metadataV4EnvVar  = "ECS_CONTAINER_METADATA_URI_V4"
	containerIDLength = 64
	defaultCgroupPath = "/proc/self/cgroup"
)

var (
	empty                    = resource.Empty()
	errCannotReadContainerID = errors.New("failed to read container ID from cGroupFile")
	errCannotReadCGroupFile  = errors.New("ECS resource detector failed to read cGroupFile")
	errNotOnECS              = errors.New("process is not on ECS, cannot detect environment variables from ECS")
)

// Create interface for methods needing to be mocked
type detectorUtils interface {
	getContainerName() (string, error)
	getContainerID() (string, error)
}

// struct implements detectorUtils interface
type ecsDetectorUtils struct{}

// resource detector collects resource information from Elastic Container Service environment
type ResourceDetector struct {
	utils detectorUtils
}

// compile time assertion that ecsDetectorUtils implements detectorUtils interface
var _ detectorUtils = (*ecsDetectorUtils)(nil)

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// Detect finds associated resources when running on ECS environment.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	metadataURIV3 := os.Getenv(metadataV3EnvVar)
	metadataURIV4 := os.Getenv(metadataV4EnvVar)

	if len(metadataURIV3) == 0 && len(metadataURIV4) == 0 {
		return empty, errNotOnECS
	}
	hostName, err := detector.utils.getContainerName()
	if err != nil {
		return empty, err
	}
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return empty, err
	}
	labels := []label.KeyValue{
		semconv.ContainerNameKey.String(hostName),
		semconv.ContainerIDKey.String(containerID),
	}

	return resource.NewWithAttributes(labels...), nil
}

// returns docker container ID from default c group path
func (ecsUtils ecsDetectorUtils) getContainerID() (string, error) {
	fileData, err := ioutil.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", errCannotReadCGroupFile
	}
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if len(str) > containerIDLength {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", errCannotReadContainerID
}

// returns host name reported by the kernel
func (ecsUtils ecsDetectorUtils) getContainerName() (string, error) {
	return os.Hostname()
}
