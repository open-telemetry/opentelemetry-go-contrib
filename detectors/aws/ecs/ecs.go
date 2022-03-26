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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	// TypeStr is AWS ECS type.
	TypeStr             = "ecs"
	metadataV3EnvVar    = "ECS_CONTAINER_METADATA_URI"
	metadataV4EnvVar    = "ECS_CONTAINER_METADATA_URI_V4"
	containerIDLength   = 64
	defaultCgroupPath   = "/proc/self/cgroup"
	ecsContainerRuntime = "docker"
	taskMetadataURIPath = "/task"
)

var (
	empty                    = resource.Empty()
	errCannotReadContainerID = errors.New("failed to read container ID from cGroupFile")
	errCannotReadCGroupFile  = errors.New("ECS resource detector failed to read cGroupFile")
)

// TaskMetadata holds the methods for ECS Task Metadata
type TaskMetadata interface {
	addMetadataToResAttributes([]attribute.KeyValue) []attribute.KeyValue
}

// Create interface for methods needing to be mocked
type detectorUtils interface {
	getContainerID() (string, error)
	getTaskMetadata(context.Context) (TaskMetadata, error)
}

// APIClient represents the network operations
type APIClient interface {
	fetch(ctx context.Context, url string) ([]byte, error)
}

type apiClient struct {
	httpClient *http.Client
}

// struct implements detectorUtils interface
type ecsDetectorUtils struct {
	apiClient APIClient
}

// resource detector collects resource information from Elastic Container Service environment
type resourceDetector struct {
	utils detectorUtils
}

// ContainerMetadataV3 represents the ECS Container Metadata in version 3 format
type ContainerMetadataV3 struct {
	DockerID   string `json:"DockerId"`
	Name       string `json:"Name"`
	DockerName string `json:"DockerName"`
	Image      string `json:"Image"`
	ImageID    string `json:"ImageID"`
}

// ContainerMetadataV4 represents the ECS Container Metadata in version 4 format
type ContainerMetadataV4 struct {
	DockerID     string `json:"DockerId"`
	Name         string `json:"Name"`
	DockerName   string `json:"DockerName"`
	Image        string `json:"Image"`
	ImageID      string `json:"ImageID"`
	ContainerARN string `json:"ContainerARN"`
}

// TaskMetadataV3 represents the ECS Task Metadata in version 3 format
type TaskMetadataV3 struct {
	Cluster          string                `json:"Cluster"`
	TaskARN          string                `json:"TaskARN"`
	Family           string                `json:"Family"`
	Revision         string                `json:"Revision"`
	AvailabilityZone string                `json:"AvailabilityZone"`
	Containers       []ContainerMetadataV3 `json:"Containers"`
}

// TaskMetadataV4 represents the ECS Task Metadata in version 4 format
type TaskMetadataV4 struct {
	Cluster          string                `json:"Cluster"`
	TaskARN          string                `json:"TaskARN"`
	Family           string                `json:"Family"`
	Revision         string                `json:"Revision"`
	AvailabilityZone string                `json:"AvailabilityZone"`
	LaunchType       string                `json:"LaunchType"`
	Containers       []ContainerMetadataV4 `json:"Containers"`
}

// compile time assertion that ecsDetectorUtils implements detectorUtils interface
var _ detectorUtils = (*ecsDetectorUtils)(nil)

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS ECS resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{utils: ecsDetectorUtils{apiClient: &apiClient{httpClient: http.DefaultClient}}}
}

// Detect finds associated resources when running on ECS environment.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	metadataURIV3 := os.Getenv(metadataV3EnvVar)
	metadataURIV4 := os.Getenv(metadataV4EnvVar)

	if len(metadataURIV3) == 0 && len(metadataURIV4) == 0 {
		return nil, nil
	}
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return empty, err
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String(containerID),
	}

	taskMetadata, err := detector.utils.getTaskMetadata(ctx)
	if err != nil {
		return empty, err
	}
	attributes = taskMetadata.addMetadataToResAttributes(attributes)
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
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
			splitData := strings.Split(str, "/")
			if len(splitData) > 0 {
				return splitData[len(splitData)-1], nil
			}
			break
		}
	}
	return "", errCannotReadContainerID
}

// returns ecs task metadata
func (ecsUtils ecsDetectorUtils) getTaskMetadata(ctx context.Context) (TaskMetadata, error) {
	metadataURIV3 := os.Getenv(metadataV3EnvVar)
	metadataURIV4 := os.Getenv(metadataV4EnvVar)

	// check of v4 endpoint first
	if metadataURIV4 != "" {
		taskMetadataURIV4 := metadataURIV4 + taskMetadataURIPath
		body, err := ecsUtils.apiClient.fetch(ctx, taskMetadataURIV4)
		if err != nil {
			return nil, err
		}
		taskMetadataV4 := TaskMetadataV4{}
		err = json.Unmarshal(body, &taskMetadataV4)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal into task metadata (v4): %w", err)
		}
		return taskMetadataV4, nil
	}
	// get metadata from v3 endpoint
	taskMetadataURIV3 := metadataURIV3 + taskMetadataURIPath
	body, err := ecsUtils.apiClient.fetch(ctx, taskMetadataURIV3)
	if err != nil {
		return nil, err
	}
	taskMetadataV3 := TaskMetadataV3{}
	err = json.Unmarshal(body, &taskMetadataV3)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal into task metadata (v3): %w", err)
	}
	return taskMetadataV3, nil
}

// adds v3 task metadata to the resource attributes
func (taskMetadataV3 TaskMetadataV3) addMetadataToResAttributes(resourceAttributes []attribute.KeyValue) []attribute.KeyValue {
	var containerID string
	for _, resAttr := range resourceAttributes {
		if resAttr.Key == semconv.ContainerIDKey {
			containerID = resAttr.Value.AsString()
			break
		}
	}
	// add container attributes
	for _, container := range taskMetadataV3.Containers {
		if container.DockerID == containerID {
			resourceAttributes = append(resourceAttributes, semconv.ContainerNameKey.String(container.DockerName))
			resourceAttributes = append(resourceAttributes, semconv.ContainerImageNameKey.String(strings.Split(container.Image, ":")[0]))
			resourceAttributes = append(resourceAttributes, semconv.ContainerImageTagKey.String(strings.Split(container.Image, ":")[1]))
			resourceAttributes = append(resourceAttributes, semconv.ContainerRuntimeKey.String(ecsContainerRuntime))
			break
		}
	}
	// add ecs task attributes
	resourceAttributes = append(resourceAttributes, semconv.AWSECSClusterARNKey.String(taskMetadataV3.Cluster))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskARNKey.String(taskMetadataV3.TaskARN))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskFamilyKey.String(taskMetadataV3.Family))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskRevisionKey.String(taskMetadataV3.Revision))

	if taskMetadataV3.AvailabilityZone != "" {
		resourceAttributes = append(resourceAttributes, semconv.CloudAvailabilityZoneKey.String(taskMetadataV3.AvailabilityZone))
	}
	return resourceAttributes
}

// adds v4 task metadata to the resource attributes
func (taskMetadataV4 TaskMetadataV4) addMetadataToResAttributes(resourceAttributes []attribute.KeyValue) []attribute.KeyValue {
	// add container attributes
	var containerID string
	for _, resAttr := range resourceAttributes {
		if resAttr.Key == semconv.ContainerIDKey {
			containerID = resAttr.Value.AsString()
			break
		}
	}
	// add container attributes
	for _, container := range taskMetadataV4.Containers {
		if container.DockerID == containerID {
			resourceAttributes = append(resourceAttributes, semconv.ContainerNameKey.String(container.DockerName))
			resourceAttributes = append(resourceAttributes, semconv.ContainerImageNameKey.String(strings.Split(container.Image, ":")[0]))
			resourceAttributes = append(resourceAttributes, semconv.ContainerImageTagKey.String(strings.Split(container.Image, ":")[1]))
			resourceAttributes = append(resourceAttributes, semconv.ContainerRuntimeKey.String(ecsContainerRuntime))
			if container.ContainerARN != "" {
				resourceAttributes = append(resourceAttributes, semconv.AWSECSContainerARNKey.String(container.ContainerARN))
			}
			break
		}
	}
	// add ecs task attributes
	resourceAttributes = append(resourceAttributes, semconv.AWSECSClusterARNKey.String(taskMetadataV4.Cluster))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskARNKey.String(taskMetadataV4.TaskARN))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskFamilyKey.String(taskMetadataV4.Family))
	resourceAttributes = append(resourceAttributes, semconv.AWSECSTaskRevisionKey.String(taskMetadataV4.Revision))

	if taskMetadataV4.AvailabilityZone != "" {
		resourceAttributes = append(resourceAttributes, semconv.CloudAvailabilityZoneKey.String(taskMetadataV4.AvailabilityZone))
	}
	if taskMetadataV4.LaunchType != "" {
		resourceAttributes = append(resourceAttributes, semconv.AWSECSLaunchtypeKey.String(taskMetadataV4.LaunchType))
	}
	return resourceAttributes
}

// fetches task metadata from the metadata endpoint
func (apiClient *apiClient) fetch(ctx context.Context, metadataURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create metadata request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := apiClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send metadata request: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read metadata response: %w", err)
	}
	defer resp.Body.Close()
	return body, nil
}
