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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// Create interface for functions that need to be mocked
type MockDetectorUtils struct {
	mock.Mock
}

// MockAPIClient mocks the methods from APIClient interface
type MockAPIClient struct {
	mock.Mock
}

func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func (detectorUtils *MockDetectorUtils) getTaskMetadata(ctx context.Context) (TaskMetadata, error) {
	args := detectorUtils.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(TaskMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (apiClient *MockAPIClient) fetch(ctx context.Context, metadataURL string) ([]byte, error) {
	args := apiClient.Called(ctx, metadataURL)
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

// successfully return resource when process is running on Amazon ECS environment
// with metadata endpoint version 3
func TestDetect_Metadata_URI_V3(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192.0.0.1/v3/43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerID").Return("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946", nil)
	detectorUtils.On("getTaskMetadata", context.Background()).Return(TaskMetadataV3{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3",
		Family:           "nginx",
		Revision:         "5",
		AvailabilityZone: "us-east-2b",
		Containers: []ContainerMetadataV3{
			{
				DockerID:   "43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946",
				Name:       "nginx-curl",
				DockerName: "ecs-nginx-5-nginx-curl-ccccb9f49db0dfe0d901",
				Image:      "nrdlngr/nginx-curl:latest",
				ImageID:    "sha256:2e00ae64383cfc865ba0a2ba37f61b50a120d2d9378559dcd458dc0de47bc165",
			},
		},
	}, nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("ecs-nginx-5-nginx-curl-ccccb9f49db0dfe0d901"),
		semconv.ContainerIDKey.String("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946"),
		semconv.ContainerImageNameKey.String("nrdlngr/nginx-curl"),
		semconv.ContainerImageTagKey.String("latest"),
		semconv.ContainerRuntimeKey.String(ecsContainerRuntime),
		semconv.AWSECSClusterARNKey.String("default"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3"),
		semconv.AWSECSTaskFamilyKey.String("nginx"),
		semconv.AWSECSTaskRevisionKey.String("5"),
		semconv.CloudAvailabilityZoneKey.String("us-east-2b"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, res, expectedResource, "Resource returned is incorrect")
	assert.NoError(t, err)
}

// successfully return resource when process is running on Amazon ECS environment
// with metadata endpoint version 4
func TestDetect_Metadata_URI_V4(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV4EnvVar, "http://192:0.0.1/v4/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerID").Return("ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca", nil)
	detectorUtils.On("getTaskMetadata", context.Background()).Return(TaskMetadataV4{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c",
		Family:           "curltest",
		Revision:         "26",
		AvailabilityZone: "us-east-2d",
		LaunchType:       "EC2",
		Containers: []ContainerMetadataV4{
			{
				DockerID:     "ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca",
				Name:         "curl",
				DockerName:   "ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00",
				Image:        "111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest:latest",
				ImageID:      "sha256:d691691e9652791a60114e67b365688d20d19940dde7c4736ea30e660d8d3553",
				ContainerARN: "arn:aws:ecs:us-west-2:111122223333:container/abb51bdd-11b4-467f-8f6c-adcfe1fe059d",
			},
		},
	}, nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String("ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca"),
		semconv.ContainerNameKey.String("ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00"),
		semconv.ContainerImageNameKey.String("111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest"),
		semconv.ContainerImageTagKey.String("latest"),
		semconv.ContainerRuntimeKey.String(ecsContainerRuntime),
		semconv.AWSECSContainerARNKey.String("arn:aws:ecs:us-west-2:111122223333:container/abb51bdd-11b4-467f-8f6c-adcfe1fe059d"),
		semconv.AWSECSClusterARNKey.String("default"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamilyKey.String("curltest"),
		semconv.AWSECSTaskRevisionKey.String("26"),
		semconv.CloudAvailabilityZoneKey.String("us-east-2d"),
		semconv.AWSECSLaunchtypeKey.String("EC2"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())
	assert.Equal(t, res, expectedResource, "Resource returned is incorrect")
	assert.NoError(t, err)
}

// returns empty resource when detector cannot read container ID
func TestDetect_CannotReadContainerID(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946")
	_ = os.Setenv(metadataV4EnvVar, "http://192:0.0.1/v4/43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00", nil)
	detectorUtils.On("getContainerID").Return("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946", errCannotReadContainerID)

	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errCannotReadContainerID, err)
	assert.Equal(t, 0, len(res.Attributes()))
}

// returns empty resource when getTaskMetadata returns error
func TestDetect_GetTaskMetadata_Returns_Err(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946")
	_ = os.Setenv(metadataV4EnvVar, "http://192:0.0.1/v4/43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00", nil)
	detectorUtils.On("getContainerID").Return("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946", nil)
	detectorUtils.On("getTaskMetadata", context.Background()).Return(nil, fmt.Errorf("could not send metadata request"))
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Error(t, err)
	assert.Equal(t, 0, len(res.Attributes()))
}

// returns empty resource when process is not running ECS
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := &resourceDetector{utils: nil}
	res, err := detector.Detect(context.Background())

	// When not on ECS, the detector should return nil and not error.
	assert.NoError(t, err, "failure to detect when not on platform must not be an error")
	assert.Nil(t, res, "failure to detect should return a nil Resource to optimize merge")
}

// returns task metadata for metadata endpoint version 3
func TestGetTaskMetadata_URI_V3(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	taskMetadataV3 := TaskMetadataV3{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c",
		Family:           "curltest",
		Revision:         "26",
		AvailabilityZone: "us-east-2d",
		Containers: []ContainerMetadataV3{
			{
				DockerID:   "ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca",
				Name:       "curl",
				DockerName: "ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00",
				Image:      "111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest:latest",
				ImageID:    "sha256:d691691e9652791a60114e67b365688d20d19940dde7c4736ea30e660d8d3553",
			},
		},
	}
	client := MockAPIClient{}
	metadataResp, _ := json.Marshal(taskMetadataV3)
	client.On("fetch", context.Background(), os.Getenv(metadataV3EnvVar)+taskMetadataURIPath).Return(metadataResp, nil)
	utils := ecsDetectorUtils{apiClient: &client}
	resp, err := utils.getTaskMetadata(context.Background())
	taskMetadataV3Resp := resp.(TaskMetadataV3)
	assert.Equal(t, taskMetadataV3, taskMetadataV3Resp)
	assert.NoError(t, err)
}

// returns nil if the task metadata fetch fails
func TestGetTaskMetadata_URI_V3_Fetch_Returns_Err(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	client := MockAPIClient{}
	client.On("fetch", context.Background(), os.Getenv(metadataV3EnvVar)+taskMetadataURIPath).Return(nil, fmt.Errorf("error in fetch"))
	utils := ecsDetectorUtils{apiClient: &client}
	resp, err := utils.getTaskMetadata(context.Background())
	assert.Nil(t, resp)
	assert.Error(t, err)
}

// returns task metadata for metadata endpoint version 4
func TestGetTaskMetadata_URI_V4(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "http://192:0.0.1/v3/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	_ = os.Setenv(metadataV4EnvVar, "http://192:0.0.1/v4/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	taskMetadataV4 := TaskMetadataV4{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c",
		Family:           "curltest",
		Revision:         "26",
		AvailabilityZone: "us-east-2d",
		LaunchType:       "EC2",
		Containers: []ContainerMetadataV4{
			{
				DockerID:     "ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca",
				Name:         "curl",
				DockerName:   "ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00",
				Image:        "111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest:latest",
				ImageID:      "sha256:d691691e9652791a60114e67b365688d20d19940dde7c4736ea30e660d8d3553",
				ContainerARN: "arn:aws:ecs:us-west-2:111122223333:container/abb51bdd-11b4-467f-8f6c-adcfe1fe059d",
			},
		},
	}
	client := MockAPIClient{}
	metadataResp, _ := json.Marshal(taskMetadataV4)
	client.On("fetch", context.Background(), os.Getenv(metadataV4EnvVar)+taskMetadataURIPath).Return(metadataResp, nil)
	utils := ecsDetectorUtils{apiClient: &client}
	resp, err := utils.getTaskMetadata(context.Background())
	taskMetadataV4Resp := resp.(TaskMetadataV4)
	assert.Equal(t, taskMetadataV4, taskMetadataV4Resp)
	assert.NoError(t, err)
}

// returns nil if the task metadata fetch fails for metadata endpoint version 4
func TestGetTaskMetadata_URI_V4_Fetch_Returns_Err(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV4EnvVar, "http://192:0.0.1/v4/ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca")
	client := MockAPIClient{}
	client.On("fetch", context.Background(), os.Getenv(metadataV4EnvVar)+taskMetadataURIPath).Return(nil, fmt.Errorf("error in fetch"))
	utils := ecsDetectorUtils{apiClient: &client}
	resp, err := utils.getTaskMetadata(context.Background())
	assert.Nil(t, resp)
	assert.Error(t, err)
}

// returns resource attributes after adding the task metadata for metadata version 3
func TestAddMetadataToResAttributes_URI_V3(t *testing.T) {
	taskMetadataV3 := TaskMetadataV3{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3",
		Family:           "nginx",
		Revision:         "5",
		AvailabilityZone: "us-east-2b",
		Containers: []ContainerMetadataV3{
			{
				DockerID:   "43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946",
				Name:       "nginx-curl",
				DockerName: "ecs-nginx-5-nginx-curl-ccccb9f49db0dfe0d901",
				Image:      "nrdlngr/nginx-curl:latest",
				ImageID:    "sha256:2e00ae64383cfc865ba0a2ba37f61b50a120d2d9378559dcd458dc0de47bc165",
			},
		},
	}
	expectedAttributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946"),
		semconv.ContainerNameKey.String("ecs-nginx-5-nginx-curl-ccccb9f49db0dfe0d901"),
		semconv.ContainerImageNameKey.String("nrdlngr/nginx-curl"),
		semconv.ContainerImageTagKey.String("latest"),
		semconv.ContainerRuntimeKey.String(ecsContainerRuntime),
		semconv.AWSECSClusterARNKey.String("default"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3"),
		semconv.AWSECSTaskFamilyKey.String("nginx"),
		semconv.AWSECSTaskRevisionKey.String("5"),
		semconv.CloudAvailabilityZoneKey.String("us-east-2b"),
	}
	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String("43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946"),
	}
	attributes = taskMetadataV3.addMetadataToResAttributes(attributes)

	assert.Equal(t, expectedAttributes, attributes)
}

// returns resource attributes after adding the task metadata for metadata version 3
func TestAddMetadataToResAttributes_URI_V4(t *testing.T) {
	taskMetadataV4 := TaskMetadataV4{
		Cluster:          "default",
		TaskARN:          "arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c",
		Family:           "curltest",
		Revision:         "26",
		AvailabilityZone: "us-east-2d",
		LaunchType:       "EC2",
		Containers: []ContainerMetadataV4{
			{
				DockerID:     "ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca",
				Name:         "curl",
				DockerName:   "ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00",
				Image:        "111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest:latest",
				ImageID:      "sha256:d691691e9652791a60114e67b365688d20d19940dde7c4736ea30e660d8d3553",
				ContainerARN: "arn:aws:ecs:us-west-2:111122223333:container/abb51bdd-11b4-467f-8f6c-adcfe1fe059d",
			},
		},
	}
	expectedAttributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String("ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca"),
		semconv.ContainerNameKey.String("ecs-curltest-26-curl-a0e7dba5aca6d8cb2e00"),
		semconv.ContainerImageNameKey.String("111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest"),
		semconv.ContainerImageTagKey.String("latest"),
		semconv.ContainerRuntimeKey.String(ecsContainerRuntime),
		semconv.AWSECSContainerARNKey.String("arn:aws:ecs:us-west-2:111122223333:container/abb51bdd-11b4-467f-8f6c-adcfe1fe059d"),
		semconv.AWSECSClusterARNKey.String("default"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamilyKey.String("curltest"),
		semconv.AWSECSTaskRevisionKey.String("26"),
		semconv.CloudAvailabilityZoneKey.String("us-east-2d"),
		semconv.AWSECSLaunchtypeKey.String("EC2"),
	}
	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerIDKey.String("ee08638adaaf009d78c248913f629e38299471d45fe7dc944d1039077e3424ca"),
	}
	attributes = taskMetadataV4.addMetadataToResAttributes(attributes)

	assert.Equal(t, expectedAttributes, attributes)
}

// returns response from the server
func TestFetch(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		_, err := resp.Write([]byte("response ok"))
		if err != nil {
			t.Errorf("Unexpected error on request: %s", err)
		}
	}))
	defer testServer.Close()
	client := apiClient{httpClient: testServer.Client()}
	data, err := client.fetch(context.Background(), testServer.URL)
	assert.Equal(t, "response ok", string(data))
	assert.NoError(t, err)
}
