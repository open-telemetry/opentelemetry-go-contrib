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
	"fmt"
	"os"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

// CloudRun collects resource information of Cloud Run instance.
type CloudRun struct{}

// compile time assertion that GCE implements the resource.Detector interface.
var _ resource.Detector = (*CloudRun)(nil)

// Detect detects associated resources when running on Cloud Run hosts.
func (cloudrun *CloudRun) Detect(ctx context.Context) (*resource.Resource, error) {
	// .OnGCE is actually testing whether the metadata server is available.
	// Metadata server is supported on Cloud Run.
	if !metadata.OnGCE() {
		return nil, nil
	}

	labels := []label.KeyValue{
		semconv.CloudProviderGCP,
	}

	var errInfo []string

	if projectID, err := metadata.ProjectID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if projectID != "" {
		labels = append(labels, semconv.CloudAccountIDKey.String(projectID))
	}

	if region, err := metadata.Get("instance/region"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if region != "" {
		labels = append(labels, semconv.CloudRegionKey.String(region))
	}

	if instanceID, err := metadata.InstanceID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if instanceID != "" {
		labels = append(labels, semconv.ServiceInstanceIDKey.String(instanceID))
	}

	// Part of Cloud Run container runtime contract.
	// See https://cloud.google.com/run/docs/reference/container-contract
	// The same K_SERVICE value ultimately maps to both `namespace` and
	// `job` label of `generic_task` metric type.
	if service := os.Getenv("K_SERVICE"); service == "" {
		errInfo = append(errInfo, "envvar K_SERVICE contains empty string.")
	} else {
		labels = append(labels,
			semconv.ServiceNamespaceKey.String(service),
			semconv.ServiceNameKey.String(service),
		)
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting Cloud Run resources: %s", errInfo)
	}

	return resource.New(labels...), aggregatedErr
}
