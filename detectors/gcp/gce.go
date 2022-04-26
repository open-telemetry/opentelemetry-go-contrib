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
	"strings"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// gceAttributes are attributes that are available from the
// GCE metadata server: https://cloud.google.com/compute/docs/metadata/default-metadata-values#vm_instance_metadata
func gceAttributes(ctx context.Context) (attributes []attribute.KeyValue, errs []string) {
	if instanceID, err := metadata.InstanceID(); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if instanceID != "" {
		attributes = append(attributes, semconv.HostIDKey.String(instanceID))
	}
	if name, err := metadata.InstanceName(); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if name != "" {
		attributes = append(attributes, semconv.HostNameKey.String(name))
	}
	if hostType, err := metadata.Get("instance/machine-type"); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if hostType != "" {
		attributes = append(attributes, semconv.HostTypeKey.String(hostType))
	}
	if zone, err := metadata.Zone(); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if zone != "" {
		attributes = append(attributes, semconv.CloudAvailabilityZoneKey.String(zone))
		splitArr := strings.SplitN(zone, "-", 3)
		if len(splitArr) == 3 {
			semconv.CloudRegionKey.String(strings.Join(splitArr[0:2], "-"))
		}
	}
	return
}
