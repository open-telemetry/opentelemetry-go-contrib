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
	tmde3EnvVar       = "ECS_CONTAINER_METADATA_URI"
	tmde4EnvVar       = "ECS_CONTAINER_METADATA_URI_V4"
	containerIDLength = 64
	defaultCgroupPath = "/proc/self/cgroup"
)

var (
	empty                    = resource.Empty()
	errCannotReadContainerId = errors.New("failed to read container ID from cGroupFile")
	errCannotReadCGroupFile  = errors.New("ECS resource detector failed to read cGroupFile")
	errNotOnECS              = errors.New("process is not on ECS, cannot detect environment variables from ECS")
)

// ecs collects resource information from Elastic Container Service environment
type ECS struct{}

// compile time assertion that AwsEksResourceDetector implements the resource.Detector interface.

var _ resource.Detector = (*ECS)(nil)

// Detect detects associated resources when running on ECS environment.
func (ecs *ECS) Detect(ctx context.Context) (*resource.Resource, error) {
	metadataUri := os.Getenv(tmde3EnvVar)
	metadataUriV4 := os.Getenv(tmde4EnvVar)

	if len(metadataUri) == 0 && len(metadataUriV4) == 0 {
		return empty, errNotOnECS
	}
	hostName, err := os.Hostname()
	if err != nil {
		return empty, err
	}
	containerID, err := getContainerID()
	if err != nil {
		return empty, err
	}
	labels := []label.KeyValue{
		semconv.ContainerNameKey.String(hostName),
		semconv.ContainerIDKey.String(containerID),
	}

	return resource.New(labels...), nil
}

func getContainerID() (string, error) {
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
	return "", errCannotReadContainerId
}
