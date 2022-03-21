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
	"strings"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// GCE collects resource information of GCE computing instances
type GCE struct {
	mc    metadataClient
	onGCE func() bool
}

var NewGCE = func() *GCE {
	return &GCE{
		mc:    metadata.NewClient(nil),
		onGCE: metadata.OnGCE,
	}
}

// compile time assertion that GCE implements the resource.Detector interface.
var _ resource.Detector = (*GCE)(nil)

// Detect detects associated resources when running on GCE hosts.
func (gce *GCE) Detect(ctx context.Context) (*resource.Resource, error) {
	if !gce.onGCE() {
		return nil, nil
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderGCP,
		semconv.CloudPlatformGCPComputeEngine,
	}

	var errInfo []string

	if projectID, err := gce.mc.ProjectID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if projectID != "" {
		attributes = append(attributes, semconv.CloudAccountIDKey.String(projectID))
	}

	if zone, err := gce.mc.Zone(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if zone != "" {
		attributes = append(attributes, semconv.CloudAvailabilityZoneKey.String(zone))

		splitArr := strings.SplitN(zone, "-", 3)
		if len(splitArr) == 3 {
			semconv.CloudRegionKey.String(strings.Join(splitArr[0:2], "-"))
		}
	}

	if instanceID, err := gce.mc.InstanceID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if instanceID != "" {
		attributes = append(attributes, semconv.HostIDKey.String(instanceID))
	}

	if name, err := gce.mc.InstanceName(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if name != "" {
		attributes = append(attributes, semconv.HostNameKey.String(name))
	}

	if hostname, err := os.Hostname(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if hostname != "" {
		attributes = append(attributes, semconv.HostNameKey.String(hostname))
	}

	if hostType, err := gce.mc.Get("instance/machine-type"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if hostType != "" {
		attributes = append(attributes, semconv.HostTypeKey.String(hostType))
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting GCE resources: %s", errInfo)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), aggregatedErr
}
