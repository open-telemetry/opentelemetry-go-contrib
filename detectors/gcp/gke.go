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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// GKE collects resource information of GKE computing instances
type GKE struct {
	mc     metadataClient
	getenv func(string) string
}

// NewCloudRun creates a CloudRun detector.
func NewGKE() *GKE {
	return &GKE{
		mc:     metadata.NewClient(nil),
		getenv: os.Getenv,
	}
}

// compile time assertion that GKE implements the resource.Detector interface.
var _ resource.Detector = (*GKE)(nil)

// Detect detects associated resources when running in GKE environment.
func (gke *GKE) Detect(ctx context.Context) (*resource.Resource, error) {
	gceDetecor := NewGCE()
	gcpLablRes, err := gceDetecor.Detect(ctx)

	if gke.getenv("KUBERNETES_SERVICE_HOST") == "" {
		return gcpLablRes, err
	}

	var errInfo []string
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderGCP,
		semconv.CloudPlatformGCPKubernetesEngine,
		semconv.K8SNamespaceNameKey.String(gke.getenv("NAMESPACE")),
		semconv.K8SPodNameKey.String(gke.getenv("HOSTNAME")),
	}

	if containerName := gke.getenv("CONTAINER_NAME"); containerName != "" {
		attributes = append(attributes, semconv.ContainerNameKey.String(containerName))
	}

	if clusterName, err := gke.mc.InstanceAttributeValue("cluster-name"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if clusterName != "" {
		attributes = append(attributes, semconv.K8SClusterNameKey.String(clusterName))
	}

	k8sattributeRes := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	res, err := resource.Merge(gcpLablRes, k8sattributeRes)
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting GKE resources: %s", errInfo)
	}

	return res, aggregatedErr
}
