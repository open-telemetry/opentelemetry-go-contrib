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
	"log"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/sdk/resource"
)

// GCE collects resource information of GCE computing instances
type GCE struct{}

// Detect detects associated resources when running on GCE hosts.
func (gce *GCE) Detect(ctx context.Context) (*resource.Resource, error) {
	if !metadata.OnGCE() {
		return nil, nil
	}

	labels := []kv.KeyValue{
		standard.CloudProviderGCP,
		standard.CloudRegionKey.String(""),
	}

	var errInfo []string

	projectID, err := metadata.ProjectID()
	logError(err)
	if projectID != "" {
		labels = append(labels, standard.CloudAccountIDKey.String(projectID))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	zone, err := metadata.Zone()
	logError(err)
	if zone != "" {
		labels = append(labels, standard.CloudZoneKey.String(zone))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	instanceID, err := metadata.InstanceID()
	logError(err)
	if instanceID != "" {
		labels = append(labels, standard.HostIDKey.String(instanceID))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	name, err := metadata.InstanceName()
	logError(err)
	if name != "" {
		labels = append(labels, standard.HostNameKey.String(name))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	hostname, err := os.Hostname()
	logError(err)
	if hostname != "" {
		labels = append(labels, standard.HostHostNameKey.String(hostname))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	hostType, err := metadata.Get("instance/machine-type")
	logError(err)
	if hostType != "" {
		labels = append(labels, standard.HostTypeKey.String(hostType))
	}
	if err != nil {
		errInfo = append(errInfo, err.Error())
	}

	var aggregatedErr error
	if len(errInfo) > 0 {
		aggregatedErr = fmt.Errorf("detecting GCE resources: %s", errInfo)
	}

	return resource.New(labels...), aggregatedErr
}

//logError logs error only if the error is present and it is not 'not defined'
func logError(err error) {
	if err != nil {
		if !strings.Contains(err.Error(), "not defined") {
			log.Printf("Error retrieving gcp metadata: %v", err)
		}
	}
}
