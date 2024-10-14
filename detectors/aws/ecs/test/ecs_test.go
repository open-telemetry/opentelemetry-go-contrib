// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ecs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	ecs "go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/stretchr/testify/assert"
)

const (
	metadataV4EnvVar = "ECS_CONTAINER_METADATA_URI_V4"
)

// successfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the EC2 Launch type.
func TestDetectV4LaunchTypeEc2(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("metadatav4-response-task-ec2.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		} else {
			content, err := os.ReadFile("metadatav4-response-container-ec2.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}))
	defer testServer.Close()

	t.Setenv(metadataV4EnvVar, testServer.URL)

	hostname, err := os.Hostname()
	assert.NoError(t, err, "Error")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.CloudAccountID("111122223333"),
		semconv.CloudRegion("us-west-2"),
		semconv.CloudAvailabilityZone("us-west-2d"),
		semconv.CloudResourceID("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.ContainerName(hostname),
		// We are not running the test in an actual container,
		// the container id is tested with mocks of the cgroup
		// file in the unit tests
		semconv.ContainerID(""),
		semconv.AWSECSContainerARN("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.AWSECSClusterARN("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("ec2"),
		semconv.AWSECSTaskARN("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamily("curltest"),
		semconv.AWSECSTaskRevision("26"),
		semconv.AWSLogGroupNames("/ecs/metadata"),
		semconv.AWSLogGroupARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:*"),
		semconv.AWSLogStreamNames("ecs/curl/8f03e41243824aea923aca126495f665"),
		semconv.AWSLogStreamARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:log-stream:ecs/curl/8f03e41243824aea923aca126495f665"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := ecs.NewResourceDetector()
	res, err := detector.Detect(context.Background())

	assert.NoError(t, err, "Detector should not fail")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// successfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the EC2 Launch type and bad ContainerARN.
func TestDetectV4LaunchTypeEc2BadContainerArn(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("metadatav4-response-task-ec2.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		} else {
			content, err := os.ReadFile("metadatav4-response-container-ec2-bad-container-arn.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}))
	defer testServer.Close()

	t.Setenv(metadataV4EnvVar, testServer.URL)

	hostname, err := os.Hostname()
	assert.NoError(t, err, "Error")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.CloudAccountID("111122223333"),
		semconv.CloudRegion("us-west-2"),
		semconv.CloudAvailabilityZone("us-west-2d"),
		semconv.CloudResourceID("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.ContainerName(hostname),
		// We are not running the test in an actual container,
		// the container id is tested with mocks of the cgroup
		// file in the unit tests
		semconv.ContainerID(""),
		semconv.AWSECSContainerARN("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.AWSECSClusterARN("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("ec2"),
		semconv.AWSECSTaskARN("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamily("curltest"),
		semconv.AWSECSTaskRevision("26"),
		semconv.AWSLogGroupNames("/ecs/metadata"),
		semconv.AWSLogGroupARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:*"),
		semconv.AWSLogStreamNames("ecs/curl/8f03e41243824aea923aca126495f665"),
		semconv.AWSLogStreamARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:log-stream:ecs/curl/8f03e41243824aea923aca126495f665"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := ecs.NewResourceDetector()
	res, err := detector.Detect(context.Background())

	assert.NoError(t, err, "Detector should not fail")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// successfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the EC2 Launch type and bad TaskARN.
func TestDetectV4LaunchTypeEc2BadTaskArn(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("metadatav4-response-task-ec2-bad-task-arn.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		} else {
			content, err := os.ReadFile("metadatav4-response-container-ec2.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}))
	defer testServer.Close()

	t.Setenv(metadataV4EnvVar, testServer.URL)

	hostname, err := os.Hostname()
	assert.NoError(t, err, "Error")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerName(hostname),
		semconv.CloudAccountID("111122223333"),
		semconv.CloudRegion("us-west-2"),
		semconv.CloudAvailabilityZone("us-west-2d"),
		semconv.CloudResourceID("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		// We are not running the test in an actual container,
		// the container id is tested with mocks of the cgroup
		// file in the unit tests
		semconv.ContainerID(""),
		semconv.AWSECSContainerARN("arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9"),
		semconv.AWSECSClusterARN("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("ec2"),
		semconv.AWSECSTaskARN("arn:aws:ecs:us-west-2:111122223333:task/default/158d1c8083dd49d6b527399fd6414f5c"),
		semconv.AWSECSTaskFamily("curltest"),
		semconv.AWSECSTaskRevision("26"),
		semconv.AWSLogGroupNames("/ecs/metadata"),
		semconv.AWSLogGroupARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:*"),
		semconv.AWSLogStreamNames("ecs/curl/8f03e41243824aea923aca126495f665"),
		semconv.AWSLogStreamARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/metadata:log-stream:ecs/curl/8f03e41243824aea923aca126495f665"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := ecs.NewResourceDetector()
	res, err := detector.Detect(context.Background())

	assert.NoError(t, err, "Detector should not fail")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// successfully returns resource when process is running on Amazon ECS environment
// with Metadata v4 with the Fargate Launch type.
func TestDetectV4LaunchTypeFargate(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.String(), "/task") {
			content, err := os.ReadFile("metadatav4-response-task-fargate.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					panic(err)
				}
			}
		} else {
			content, err := os.ReadFile("metadatav4-response-container-fargate.json")
			if err == nil {
				_, err = res.Write(content)
				if err != nil {
					panic(err)
				}
			}
		}
	}))
	defer testServer.Close()

	t.Setenv(metadataV4EnvVar, testServer.URL)

	hostname, err := os.Hostname()
	assert.NoError(t, err, "Error")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerName(hostname),
		semconv.CloudAccountID("111122223333"),
		semconv.CloudRegion("us-west-2"),
		semconv.CloudAvailabilityZone("us-west-2a"),
		semconv.CloudResourceID("arn:aws:ecs:us-west-2:111122223333:container/05966557-f16c-49cb-9352-24b3a0dcd0e1"),
		// We are not running the test in an actual container,
		// the container id is tested with mocks of the cgroup
		// file in the unit tests
		semconv.ContainerID(""),
		semconv.AWSECSContainerARN("arn:aws:ecs:us-west-2:111122223333:container/05966557-f16c-49cb-9352-24b3a0dcd0e1"),
		semconv.AWSECSClusterARN("arn:aws:ecs:us-west-2:111122223333:cluster/default"),
		semconv.AWSECSLaunchtypeKey.String("fargate"),
		semconv.AWSECSTaskARN("arn:aws:ecs:us-west-2:111122223333:task/default/e9028f8d5d8e4f258373e7b93ce9a3c3"),
		semconv.AWSECSTaskFamily("curltest"),
		semconv.AWSECSTaskRevision("3"),
		semconv.AWSLogGroupNames("/ecs/containerlogs"),
		semconv.AWSLogGroupARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/containerlogs:*"),
		semconv.AWSLogStreamNames("ecs/curl/cd189a933e5849daa93386466019ab50"),
		semconv.AWSLogStreamARNs("arn:aws:logs:us-west-2:111122223333:log-group:/ecs/containerlogs:log-stream:ecs/curl/cd189a933e5849daa93386466019ab50"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := ecs.NewResourceDetector()
	res, err := detector.Detect(context.Background())

	assert.NoError(t, err, "Detector should not fail")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}
