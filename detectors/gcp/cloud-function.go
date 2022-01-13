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

package gcp

import (
	"context"
	"errors"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var (
	errNotOnGoogleCloudFunction = errors.New("cannot detect environment variables from Google Cloud Function")
)

const (
	gcpFunctionNameKey = "K_SERVICE"
)

//NewResourceDetector will return an implementation for gcp cloud function resource detector
func NewResourceDetector() resource.Detector {
	return &CloudFunction{
		client: &gcpClientImpl{},
	}
}

type gcpClient interface {
	gcpProjectID() (string, error)
	gcpRegion() (string, error)
}
type gcpClientImpl struct{}

func (gi *gcpClientImpl) gcpProjectID() (string, error) {
	return metadata.ProjectID()
}

func (gi *gcpClientImpl) gcpRegion() (string, error) {
	var region string
	zone, err := metadata.Zone()
	if zone != "" {
		splitArr := strings.SplitN(zone, "-", 3)
		if len(splitArr) == 3 {
			region = strings.Join(splitArr[0:2], "-")
		}
	}
	return region, err
}

type CloudFunction struct {
	client gcpClient
}

// Detect detects associated resources when running in  cloud function.
func (f *CloudFunction) Detect(ctx context.Context) (*resource.Resource, error) {
	functionName, ok := f.googleCloudFunctionName()
	if !ok {
		return nil, errNotOnGoogleCloudFunction
	}

	projectID, err := f.client.gcpProjectID()
	if err != nil {
		return nil, err
	}
	region, err := f.client.gcpRegion()
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderGCP,
		semconv.CloudPlatformGCPCloudFunctions,
		attribute.String(string(semconv.FaaSNameKey), functionName),
		semconv.CloudAccountIDKey.String(projectID),
		semconv.CloudRegionKey.String(region),
	}
	return resource.NewSchemaless(attributes...), nil

}

func (f *CloudFunction) googleCloudFunctionName() (string, bool) {
	return os.LookupEnv(gcpFunctionNameKey)
}
