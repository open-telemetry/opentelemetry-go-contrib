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

package gcp // import "go.opentelemetry.io/contrib/detectors/gcp"

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// Detector collects resource information for GCP platforms
type Detector struct{}

// compile time assertion that Detector implements the resource.Detector interface.
var _ resource.Detector = (*Detector)(nil)

func (d *Detector) Detect(ctx context.Context) (*resource.Resource, error) {
	if !metadata.OnGCE() {
		return nil, nil
	}
	attributes := []attribute.KeyValue{semconv.CloudProviderGCP}

	var errors []string
	attrs, errs := projectAttributes(ctx)
	attributes = append(attributes, attrs...)
	errors = append(errors, errs...)

	if onGKE() {
		attributes = append(attributes, semconv.CloudPlatformGCPKubernetesEngine)
		attrs, errs = gceAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
		attrs, errs = gkeAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
	} else if onCloudRun() {
		attributes = append(attributes, semconv.CloudPlatformGCPCloudRun)
		attrs, errs = faasAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
	} else if onCloudFunctions() {
		attributes = append(attributes, semconv.CloudPlatformGCPCloudFunctions)
		attrs, errs = faasAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
	} else if onAppEngine() {
		attributes = append(attributes, semconv.CloudPlatformGCPAppEngine)
		attrs, errs = appEngineAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
	} else {
		attributes = append(attributes, semconv.CloudPlatformGCPComputeEngine)
		attrs, errs = gceAttributes(ctx)
		attributes = append(attributes, attrs...)
		errors = append(errors, errs...)
	}
	var aggregatedErr error
	if len(errors) > 0 {
		aggregatedErr = fmt.Errorf("detecting GCP resources: %s", errors)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), aggregatedErr
}

func projectAttributes(ctx context.Context) (attributes []attribute.KeyValue, errs []string) {
	if projectID, err := metadata.ProjectID(); err != nil {
		errs = append(errs, err.Error())
	} else if projectID != "" {
		attributes = append(attributes, semconv.CloudAccountIDKey.String(projectID))
	}
	return
}

// hasProblem checks if the err is not nil or for missing resources
func hasProblem(err error) bool {
	if err == nil {
		return false
	}
	if _, undefined := err.(metadata.NotDefinedError); undefined {
		return false
	}
	return true
}
