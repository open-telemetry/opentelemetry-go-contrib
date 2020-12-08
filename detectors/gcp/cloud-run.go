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

const serviceNamespace = "cloud-run-managed"

// The minimal list of metadata.Client methods we use. Use an interface so we
// can replace it with a fake implementation in the unit test.
type metadataClient interface {
	ProjectID() (string, error)
	Get(string) (string, error)
	InstanceID() (string, error)
}

// CloudRun collects resource information of Cloud Run instance.
type CloudRun struct {
	mc     metadataClient
	onGCE  func() bool
	getenv func(string) string
}

// compile time assertion that CloudRun implements the resource.Detector
// interface.
var _ resource.Detector = (*CloudRun)(nil)

// NewCloudRun creates a CloudRun detector.
func NewCloudRun() *CloudRun {
	return &CloudRun{
		mc:     metadata.NewClient(nil),
		onGCE:  metadata.OnGCE,
		getenv: os.Getenv,
	}
}

// Detect detects associated resources when running on Cloud Run hosts.
// NOTE: the service.namespace label is currently hardcoded to be
// "cloud-run-managed". This may change in the future, please do not rely on
// this behavior yet.
func (c *CloudRun) Detect(ctx context.Context) (*resource.Resource, error) {
	// .OnGCE is actually testing whether the metadata server is available.
	// Metadata server is supported on Cloud Run.
	if !c.onGCE() {
		return nil, nil
	}

	labels := []label.KeyValue{
		semconv.CloudProviderGCP,
		semconv.ServiceNamespaceKey.String(serviceNamespace),
	}

	var errInfo []string

	if projectID, err := c.mc.ProjectID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if projectID != "" {
		labels = append(labels, semconv.CloudAccountIDKey.String(projectID))
	}

	if region, err := c.mc.Get("instance/region"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if region != "" {
		labels = append(labels, semconv.CloudRegionKey.String(region))
	}

	if instanceID, err := c.mc.InstanceID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if instanceID != "" {
		labels = append(labels, semconv.ServiceInstanceIDKey.String(instanceID))
	}

	// Part of Cloud Run container runtime contract.
	// See https://cloud.google.com/run/docs/reference/container-contract
	if service := c.getenv("K_SERVICE"); service == "" {
		errInfo = append(errInfo, "envvar K_SERVICE contains empty string.")
	} else {
		labels = append(labels, semconv.ServiceNameKey.String(service))
	}
	resource, err := resource.New(ctx, resource.WithAttributes(labels...))
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting Cloud Run resources: %s", errInfo)
	}

	return resource, aggregatedErr
}
