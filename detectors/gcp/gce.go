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

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

// GCE collects resource information of GCE computing instances
type GCE struct{}

// compile time assertion that GCE implements the resource.Detector interface.
var _ resource.Detector = (*GCE)(nil)

// Detect detects associated resources when running on GCE hosts.
func (gce *GCE) Detect(ctx context.Context) (*resource.Resource, error) {
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

	if zone, err := metadata.Zone(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if zone != "" {
		labels = append(labels, semconv.CloudZoneKey.String(zone))

		splitArr := strings.SplitN(zone, "-", 3)
		if len(splitArr) == 3 {
			semconv.CloudRegionKey.String(strings.Join(splitArr[0:2], "-"))
		}
	}

	if instanceID, err := metadata.InstanceID(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if instanceID != "" {
		labels = append(labels, semconv.HostIDKey.String(instanceID))
	}

	if name, err := metadata.InstanceName(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if name != "" {
		labels = append(labels, semconv.HostNameKey.String(name))
	}

	if hostname, err := os.Hostname(); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if hostname != "" {
		labels = append(labels, semconv.HostHostNameKey.String(hostname))
	}

	if hostType, err := metadata.Get("instance/machine-type"); hasProblem(err) {
		errInfo = append(errInfo, err.Error())
	} else if hostType != "" {
		labels = append(labels, semconv.HostTypeKey.String(hostType))
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting GCE resources: %s", errInfo)
	}

	return resource.New(labels...), aggregatedErr
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
