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
	http "net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ecsmetadata "github.com/brunoscheufler/aws-ecs-metadata-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// Create interface for functions that need to be mocked.
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

// succesfully returns resource when process is running on Amazon ECS environment
// with no Metadata v4.
func TestDetectV3(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")

	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, _ := detector.Detect(context.Background())

	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// succesfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the EC2 Launch type.
func TestDetectV4LaunchTypeEc2(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("testdata/metadatav4-response-task-ec2.json")
			if err == nil {
				res.Write(content)
			}
		} else {
			content, err := os.ReadFile("testdata/metadatav4-response-container-ec2.json")
			if err == nil {
				res.Write(content)
			}
		}
	}))
	defer testServer.Close()

	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	_ = os.Setenv(metadataV4EnvVar, testServer.URL)

	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
		semconv.AWSECSContainerARNKey.String("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.AWSECSClusterARNKey.String("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("ec2"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamilyKey.String("curltest"),
		semconv.AWSECSTaskRevisionKey.String("26"),
		semconv.AWSLogGroupNamesKey.String("/ecs/metadata"),
		semconv.AWSLogGroupARNsKey.String("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:*"),
		semconv.AWSLogStreamNamesKey.String("ecs/curl/8f03e41243824aea923aca126495f665"),
		semconv.AWSLogStreamARNsKey.String("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:log-stream:ecs/curl/8f03e41243824aea923aca126495f665"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, err, nil, "Detector should not file")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// succesfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the Fargate Launch type.
func TestDetectV4LaunchTypeFargate(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("testdata/metadatav4-response-task-fargate.json")
			if err == nil {
				res.Write(content)
			}
		} else {
			content, err := os.ReadFile("testdata/metadatav4-response-container-fargate.json")
			if err == nil {
				res.Write(content)
			}
		}
	}))
	defer testServer.Close()

	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	_ = os.Setenv(metadataV4EnvVar, testServer.URL)

	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
		semconv.AWSECSContainerARNKey.String("arn:aws:ecs:us-west-2:111122223333:container/05966557-f16c-49cb-9352-24b3a0dcd0e1"),
		semconv.AWSECSClusterARNKey.String("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("fargate"),
		semconv.AWSECSTaskARNKey.String("arn:aws:ecs:us-west-2:111122223333:task/default/e9028f8d5d8e4f258373e7b93ce9a3c3"),
		semconv.AWSECSTaskFamilyKey.String("curltest"),
		semconv.AWSECSTaskRevisionKey.String("3"),
		semconv.AWSLogGroupNamesKey.String("/ecs/containerlogs"),
		semconv.AWSLogGroupARNsKey.String("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/containerlogs:*"),
		semconv.AWSLogStreamNamesKey.String("ecs/curl/cd189a933e5849daa93386466019ab50"),
		semconv.AWSLogStreamARNsKey.String("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/containerlogs:log-stream:ecs/curl/cd189a933e5849daa93386466019ab50"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, err, nil, "Detector should not file")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// returns empty resource when detector cannot read container ID.
func TestDetectCannotReadContainerID(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	_ = os.Setenv(metadataV4EnvVar, "4")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("", errCannotReadContainerID)

	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errCannotReadContainerID, err)
	assert.Equal(t, 0, len(res.Attributes()))
}

// returns empty resource when detector cannot read container Name.
func TestDetectCannotReadContainerName(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	_ = os.Setenv(metadataV4EnvVar, "4")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("", errCannotReadContainerName)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errCannotReadContainerName, err)
	assert.Equal(t, 0, len(res.Attributes()))
}

func TestDetectV4Parse(t *testing.T) {
	content, _ := os.ReadFile("testdata/metadatav4-response-task.json")

	taskMetadata := &ecsmetadata.TaskMetadataV4{}
	json.Unmarshal(content, taskMetadata)
}

// returns empty resource when process is not running ECS.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := &resourceDetector{utils: nil}
	res, err := detector.Detect(context.Background())

	// When not on ECS, the detector should return nil and not error.
	assert.NoError(t, err, "failure to detect when not on platform must not be an error")
	assert.Nil(t, res, "failure to detect should return a nil Resource to optimize merge")
}
