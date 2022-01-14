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
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	gcpFunctionNameKey = "K_SERVICE"
)

//NewCloudFunction will return an implementation for gcp cloud function resource detector
func NewCloudFunction() resource.Detector {
	return &CloudFunction{
		cloudRun: NewCloudRun(),
	}
}

type CloudFunction struct {
	cloudRun *CloudRun
}

// Detect detects associated resources when running in  cloud function.
func (f *CloudFunction) Detect(ctx context.Context) (*resource.Resource, error) {
	functionName, ok := f.googleCloudFunctionName()
	if !ok {
		return nil, nil
	}

	projectID, err := f.cloudRun.mc.ProjectID()
	if err != nil {
		return nil, err
	}
	region, err := f.cloudRun.cloudRegion(ctx)
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
