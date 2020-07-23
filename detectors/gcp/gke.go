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

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/sdk/resource"
)

// GKE collects resource information of GKE computing instances
type GKE struct{}

// compile time assertion that GCE implements the resource.Detector interface.
var _ resource.Detector = (*GKE)(nil)

// Detect detects associated resources when running in GKE environment.
func (gke *GKE) Detect(ctx context.Context) (*resource.Resource, error) {
	gcpDetecor := GCE{}
	gceLablRes, err := gcpDetecor.Detect(ctx)

	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		return gceLablRes, err
	}

	var errInfo []string
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	labels := []kv.KeyValue{
		standard.K8SNamespaceNameKey.String(os.Getenv("NAMESPACE")),
		standard.K8SPodNameKey.String(os.Getenv("HOSTNAME")),
	}

	if containerName := os.Getenv("CONTAINER_NAME"); containerName != "" {
		labels = append(labels, standard.ContainerNameKey.String(containerName))
	}

	if clusterName, err := metadata.Get("cluster-name"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if clusterName != "" {
		labels = append(labels, standard.K8SClusterNameKey.String(clusterName))
	}

	k8sLabelRes := resource.New(labels...)
	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting GCE resources: %s", errInfo)
	}

	return resource.Merge(gceLablRes, k8sLabelRes), aggregatedErr
}
