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
	"os"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func onGKE() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// gkeAttributes detects resource attributes available via the GKE Metadata server:
// https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity#instance_attributes
func gkeAttributes(ctx context.Context) (attributes []attribute.KeyValue, errs []string) {
	if clusterName, err := metadata.InstanceAttributeValue("cluster-name"); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if clusterName != "" {
		attributes = append(attributes, semconv.K8SClusterNameKey.String(clusterName))
	}
	// TODO: with workload identity enabled, do we need to use cluster-location instead of the GCE availability zone?
	return
}
