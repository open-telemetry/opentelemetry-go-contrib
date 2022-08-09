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

package ecs // import "go.opentelemetry.io/contrib/detectors/aws/ecs"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	ecsmetadata "github.com/brunoscheufler/aws-ecs-metadata-go"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const (
	// TypeStr is AWS ECS type.
	TypeStr           = "ecs"
	metadataV3EnvVar  = "ECS_CONTAINER_METADATA_URI"
	metadataV4EnvVar  = "ECS_CONTAINER_METADATA_URI_V4"
	containerIDLength = 64
	defaultCgroupPath = "/proc/self/cgroup"
)

var (
	empty                                 = resource.Empty()
	errCannotReadContainerID              = errors.New("failed to read container ID from cGroupFile")
	errCannotReadContainerName            = errors.New("failed to read hostname")
	errCannotReadCGroupFile               = errors.New("ECS resource detector failed to read cGroupFile")
	errCannotRetrieveMetadataV4           = errors.New("ECS resource detector failed to retrieve metadata from the ECS Metadata v4 container endpoint")
	errCannotRetrieveMetadataV4Task       = errors.New("ECS resource detector failed to retrieve metadata from the ECS Metadata v4 task endpoint")
	errCannotRetrieveLogsGroupMetadataV4  = errors.New("The ECS Metadata v4 did not return a AwsLogGroup name")
	errCannotRetrieveLogsStreamMetadataV4 = errors.New("The ECS Metadata v4 did not return a AwsLogStream name")
	errCannotParseAwsRegionMetadataV4     = errors.New("Cannot parse the AWS region our of the Container ARN returned by the ECS Metadata v4 container endpoint")
	errCannotParseAwsAccountMetadataV4    = errors.New("Cannot parse the AWS account our of the Container ARN returned by the ECS Metadata v4 container endpoint")
)

// Create interface for methods needing to be mocked.
type detectorUtils interface {
	getContainerName() (string, error)
	getContainerID() (string, error)
}

// struct implements detectorUtils interface.
type ecsDetectorUtils struct{}

// resource detector collects resource information from Elastic Container Service environment.
type resourceDetector struct {
	utils detectorUtils
}

// compile time assertion that ecsDetectorUtils implements detectorUtils interface.
var _ detectorUtils = (*ecsDetectorUtils)(nil)

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS ECS resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{
		utils: ecsDetectorUtils{},
	}
}

// Detect finds associated resources when running on ECS environment.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	metadataURIV3 := os.Getenv(metadataV3EnvVar)
	metadataURIV4 := os.Getenv(metadataV4EnvVar)

	if len(metadataURIV3) == 0 && len(metadataURIV4) == 0 {
		return nil, nil
	}
	hostName, err := detector.utils.getContainerName()
	if err != nil {
		return empty, err
	}
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return empty, err
	}
	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String(hostName),
		semconv.ContainerIDKey.String(containerID),
	}

	if len(metadataURIV4) > 0 {
		containerMetadata, err := ecsmetadata.GetContainerV4(ctx, &http.Client{})
		if err != nil {
			return empty, err
		}
		attributes = append(
			attributes,
			semconv.AWSECSContainerARNKey.String(containerMetadata.ContainerARN),
		)

		taskMetadata, err := ecsmetadata.GetTaskV4(ctx, &http.Client{})
		if err != nil {
			return empty, err
		}

		clusterArn := taskMetadata.Cluster
		if !strings.HasPrefix(clusterArn, "arn:") {
			baseArn := containerMetadata.ContainerARN[:strings.LastIndex(containerMetadata.ContainerARN, ":")]
			clusterArn = fmt.Sprintf("%s:cluster/%s", baseArn, clusterArn)
		}

		logAttributes, err := detector.getLogsAttributes(containerMetadata)
		if err != nil {
			return empty, err
		}

		if len(logAttributes) > 0 {
			attributes = append(attributes, logAttributes...)
		}

		attributes = append(
			attributes,
			semconv.AWSECSClusterARNKey.String(clusterArn),
			semconv.AWSECSLaunchtypeKey.String(strings.ToLower(taskMetadata.LaunchType)),
			semconv.AWSECSTaskARNKey.String(taskMetadata.TaskARN),
			semconv.AWSECSTaskFamilyKey.String(taskMetadata.Family),
			semconv.AWSECSTaskRevisionKey.String(taskMetadata.Revision),
		)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *resourceDetector) getLogsAttributes(metadata *ecsmetadata.ContainerMetadataV4) ([]attribute.KeyValue, error) {
	if metadata.LogDriver != "awslogs" {
		return []attribute.KeyValue{}, nil
	}

	logsOptions := metadata.LogOptions

	if len(logsOptions.AwsLogsGroup) < 1 {
		return nil, errCannotRetrieveLogsGroupMetadataV4
	}

	if len(logsOptions.AwsLogsStream) < 1 {
		return nil, errCannotRetrieveLogsStreamMetadataV4
	}

	containerArn := metadata.ContainerARN
	logsRegion := logsOptions.AwsRegion
	if len(logsRegion) < 1 {
		r := regexp.MustCompile(`arn:aws:ecs:([^:]+):.*`)
		logsRegion = r.FindStringSubmatch(containerArn)[1]
	}

	r := regexp.MustCompile(`arn:aws:ecs:[^:]+:([^:]+):.*`)
	awsAccount := r.FindStringSubmatch(containerArn)[1]

	return []attribute.KeyValue{
		semconv.AWSLogGroupNamesKey.String(logsOptions.AwsLogsGroup),
		semconv.AWSLogGroupARNsKey.String(fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s:*", logsRegion, awsAccount, logsOptions.AwsLogsGroup)),
		semconv.AWSLogStreamNamesKey.String(logsOptions.AwsLogsStream),
		semconv.AWSLogStreamARNsKey.String(fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s:log-stream:%s", logsRegion, awsAccount, logsOptions.AwsLogsGroup, logsOptions.AwsLogsStream)),
	}, nil
}

// returns docker container ID from default c group path.
func (ecsUtils ecsDetectorUtils) getContainerID() (string, error) {
	fileData, err := os.ReadFile(defaultCgroupPath)
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

// returns host name reported by the kernel.
func (ecsUtils ecsDetectorUtils) getContainerName() (string, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return "", errCannotReadContainerName
	}
	return hostName, nil
}
