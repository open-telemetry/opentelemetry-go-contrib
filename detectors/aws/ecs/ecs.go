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
	"runtime"
	"strings"

	ecsmetadata "github.com/brunoscheufler/aws-ecs-metadata-go"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
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
	errCannotReadContainerName            = errors.New("failed to read hostname")
	errCannotRetrieveAccountID            = errors.New("cannot parse the account ID from the ARN")
	errCannotRetrieveRegion               = errors.New("cannot parse the region from the ARN")
	errCannotRetrieveLogsGroupMetadataV4  = errors.New("the ECS Metadata v4 did not return a AwsLogGroup name")
	errCannotRetrieveLogsStreamMetadataV4 = errors.New("the ECS Metadata v4 did not return a AwsLogStream name")
)

// Create interface for methods needing to be mocked.
type detectorUtils interface {
	getContainerName() (string, error)
	getContainerID() (string, error)
	getContainerMetadataV4(ctx context.Context) (*ecsmetadata.ContainerMetadataV4, error)
	getTaskMetadataV4(ctx context.Context) (*ecsmetadata.TaskMetadataV4, error)
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
		semconv.ContainerName(hostName),
		semconv.ContainerID(containerID),
	}

	if len(metadataURIV4) > 0 {
		containerMetadata, err := detector.utils.getContainerMetadataV4(ctx)
		if err != nil {
			return empty, err
		}

		taskMetadata, err := detector.utils.getTaskMetadataV4(ctx)
		if err != nil {
			return empty, err
		}

		baseArn := detector.getBaseArn(
			taskMetadata.TaskARN,
			containerMetadata.ContainerARN,
			taskMetadata.Cluster,
		)

		if baseArn != "" {
			if !strings.HasPrefix(taskMetadata.Cluster, "arn:") {
				taskMetadata.Cluster = fmt.Sprintf("%s:cluster/%s", baseArn, taskMetadata.Cluster)
			}
			if !strings.HasPrefix(containerMetadata.ContainerARN, "arn:") {
				containerMetadata.ContainerARN = fmt.Sprintf("%s:container/%s", baseArn, containerMetadata.ContainerARN)
			}
			if !strings.HasPrefix(taskMetadata.TaskARN, "arn:") {
				taskMetadata.TaskARN = fmt.Sprintf("%s:task/%s", baseArn, taskMetadata.TaskARN)
			}
		}

		if accountId, err := detector.getAccountID(taskMetadata.TaskARN); err == nil {
			attributes = append(
				attributes,
				semconv.CloudAccountID(accountId),
			)
		}

		if region, err := detector.getRegion(taskMetadata.TaskARN); err == nil {
			attributes = append(
				attributes,
				semconv.CloudRegion(region),
			)
		}

		availabilityZone := taskMetadata.AvailabilityZone
		if len(availabilityZone) > 0 {
			attributes = append(
				attributes,
				semconv.CloudAvailabilityZone(availabilityZone),
			)
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
			semconv.AWSECSContainerARN(containerMetadata.ContainerARN),
			semconv.AWSECSClusterARN(taskMetadata.Cluster),
			semconv.AWSECSLaunchtypeKey.String(strings.ToLower(taskMetadata.LaunchType)),
			semconv.AWSECSTaskARN(taskMetadata.TaskARN),
			semconv.AWSECSTaskFamily(taskMetadata.Family),
			semconv.AWSECSTaskRevision(taskMetadata.Revision),
		)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *resourceDetector) getBaseArn(arns ...string) string {
	for _, arn := range arns {
		if i := strings.LastIndex(arn, ":"); i >= 0 {
			return arn[:i]
		}
	}
	return ""
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
	// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	const arnPartition = 1
	const arnRegion = 3
	const arnAccountId = 4
	containerArnParts := strings.Split(containerArn, ":")
	// a valid arn should have at least 6 parts
	if len(containerArnParts) < 6 {
		return nil, errCannotRetrieveLogsStreamMetadataV4
	}
	logsRegion := logsOptions.AwsRegion
	if len(logsRegion) < 1 {
		logsRegion = containerArnParts[arnRegion]
	}

	awsPartition := containerArnParts[arnPartition]
	awsAccount := containerArnParts[arnAccountId]

	awsLogGroupArn := strings.Join([]string{
		"arn", awsPartition, "logs",
		logsRegion, awsAccount, "log-group", logsOptions.AwsLogsGroup,
		"*",
	}, ":")
	awsLogStreamArn := strings.Join([]string{
		"arn", awsPartition, "logs",
		logsRegion, awsAccount, "log-group", logsOptions.AwsLogsGroup,
		"log-stream", logsOptions.AwsLogsStream,
	}, ":")

	return []attribute.KeyValue{
		semconv.AWSLogGroupNames(logsOptions.AwsLogsGroup),
		semconv.AWSLogGroupARNs(awsLogGroupArn),
		semconv.AWSLogStreamNames(logsOptions.AwsLogsStream),
		semconv.AWSLogStreamARNs(awsLogStreamArn),
	}, nil
}

// returns the AWS account ID from an ARN.
func (detector *resourceDetector) getAccountID(arn string) (string, error) {
	arnParts := strings.Split(arn, ":")
	// a valid arn should have at least 6 parts
	if len(arnParts) < 6 {
		return "", errCannotRetrieveAccountID
	}
	return arnParts[4], nil
}

// returns the AWS region from an ARN.
func (detector *resourceDetector) getRegion(arn string) (string, error) {
	arnParts := strings.Split(arn, ":")
	// a valid arn should have at least 6 parts
	if len(arnParts) < 6 {
		return "", errCannotRetrieveRegion
	}
	return arnParts[3], nil
}

// returns metadata v4 for the container.
func (ecsUtils ecsDetectorUtils) getContainerMetadataV4(ctx context.Context) (*ecsmetadata.ContainerMetadataV4, error) {
	return ecsmetadata.GetContainerV4(ctx, &http.Client{})
}

// returns metadata v4 for the task.
func (ecsUtils ecsDetectorUtils) getTaskMetadataV4(ctx context.Context) (*ecsmetadata.TaskMetadataV4, error) {
	return ecsmetadata.GetTaskV4(ctx, &http.Client{})
}

// returns docker container ID from default c group path.
func (ecsUtils ecsDetectorUtils) getContainerID() (string, error) {
	if runtime.GOOS != "linux" {
		// Cgroups are used only under Linux.
		return "", nil
	}

	fileData, err := os.ReadFile(defaultCgroupPath)
	if err != nil {
		// Cgroups file not found.
		// For example, windows; or when running integration tests outside of a container.
		return "", nil
	}
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if len(str) > containerIDLength {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", nil
}

// returns host name reported by the kernel.
func (ecsUtils ecsDetectorUtils) getContainerName() (string, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return "", errCannotReadContainerName
	}
	return hostName, nil
}
