// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2/google"
)

const (
	// See the available GCE instance metadata:
	// https://cloud.google.com/compute/docs/metadata/predefined-metadata-keys#instance-metadata
	machineTypeMetadataAttr = "instance/machine-type"
	// https://cloud.google.com/compute/docs/instance-groups/getting-info-about-migs#checking_if_a_vm_instance_is_part_of_a_mig
	createdByInstanceAttr    = "created-by"
	computeInstancesGetScope = "https://www.googleapis.com/auth/compute.readonly"
	defaultComputeBaseURL    = "https://compute.googleapis.com/compute/v1"
)

func (d *Detector) onGCE(ctx context.Context) bool {
	_, err := d.metadata.GetWithContext(ctx, machineTypeMetadataAttr)
	return err == nil
}

// GCEHostType returns the machine type of the instance on which this program is running.
func (d *Detector) GCEHostType(ctx context.Context) (string, error) {
	return d.metadata.GetWithContext(ctx, machineTypeMetadataAttr)
}

// GCEHostID returns the instance ID of the instance on which this program is running.
func (d *Detector) GCEHostID(ctx context.Context) (string, error) {
	return d.instanceID(ctx)
}

// GCEHostName returns the instance name of the instance on which this program is running.
// Recommended to use GCEInstanceName() or GCEInstanceHostname() to more accurately reflect which
// value is returned.
func (d *Detector) GCEHostName(ctx context.Context) (string, error) {
	return d.metadata.InstanceNameWithContext(ctx)
}

// GCEInstanceName returns the instance name of the instance on which this program is running.
// This is the value visible in the Cloud Console UI, and the prefix for the default hostname
// of the instance as defined by the default internal DNS name (see https://cloud.google.com/compute/docs/internal-dns#instance-fully-qualified-domain-names).
func (d *Detector) GCEInstanceName(ctx context.Context) (string, error) {
	return d.metadata.InstanceNameWithContext(ctx)
}

// GCEInstanceHostname returns the full value of the default or custom hostname of the instance
// on which this program is running. See https://cloud.google.com/compute/docs/instances/custom-hostname-vm.
func (d *Detector) GCEInstanceHostname(ctx context.Context) (string, error) {
	return d.metadata.HostnameWithContext(ctx)
}

// GCEAvailabilityZoneAndRegion returns the zone and region in which this program is running.
func (d *Detector) GCEAvailabilityZoneAndRegion(ctx context.Context) (string, string, error) {
	zone, err := d.metadata.ZoneWithContext(ctx)
	if err != nil {
		return "", "", err
	}
	if zone == "" {
		return "", "", errors.New("no zone detected from GCE metadata server")
	}
	splitZone := strings.SplitN(zone, "-", 3)
	if len(splitZone) != 3 {
		return "", "", fmt.Errorf("zone was not in the expected format: country-region-zone.  Got %v", zone)
	}
	return zone, strings.Join(splitZone[0:2], "-"), nil
}

type ManagedInstanceGroup struct {
	Name     string
	Location string
	Type     LocationType
}

var createdByMIGRE = regexp.MustCompile(`^projects/[^/]+/(zones|regions)/([^/]+)/instanceGroupManagers/([^/]+)$`)

func (d *Detector) GCEManagedInstanceGroup(ctx context.Context) (ManagedInstanceGroup, error) {
	createdBy, err := d.metadata.InstanceAttributeValueWithContext(ctx, createdByInstanceAttr)
	if _, ok := errors.AsType[metadata.NotDefinedError](err); ok {
		return ManagedInstanceGroup{}, nil
	} else if err != nil {
		return ManagedInstanceGroup{}, err
	}
	matches := createdByMIGRE.FindStringSubmatch(createdBy)
	if matches == nil {
		// The "created-by" key exists, but it doesn't describe a MIG.
		// Something else must have created this VM.
		return ManagedInstanceGroup{}, nil
	}

	mig := ManagedInstanceGroup{
		Name:     matches[3],
		Location: matches[2],
	}
	switch matches[1] {
	case "zones":
		mig.Type = Zone
	case "regions":
		mig.Type = Region
	}
	return mig, nil
}

type computeInstance struct {
	MachineType string            `json:"machineType"`
	Labels      map[string]string `json:"labels"`
}

// GCEInstanceLabels returns the labels of the GCE instance on which this program is running.
func (d *Detector) GCEInstanceLabels(ctx context.Context) (map[string]string, error) {
	instance, err := d.computeInstance(ctx)
	if err != nil {
		return nil, err
	}
	return instance.Labels, nil
}

func (d *Detector) computeInstance(ctx context.Context) (*computeInstance, error) {
	projectID, err := d.metadata.GetWithContext(ctx, "project/project-id")
	if err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	zone, err := d.metadata.ZoneWithContext(ctx)
	if err != nil {
		return nil, err
	}
	instanceName, err := d.metadata.InstanceNameWithContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := d.computeHTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := d.computeBaseURL
	if baseURL == "" {
		baseURL = defaultComputeBaseURL
	}
	reqURL := fmt.Sprintf("%s/projects/%s/zones/%s/instances/%s",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(projectID),
		url.PathEscape(lastPathSegment(zone)),
		url.PathEscape(instanceName),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("compute instances.get returned %s", res.Status)
	}
	var instance computeInstance
	if err := json.NewDecoder(res.Body).Decode(&instance); err != nil {
		return nil, err
	}
	return &instance, nil
}

func (d *Detector) computeHTTPClient(ctx context.Context) (*http.Client, error) {
	if d.httpClient != nil {
		return d.httpClient, nil
	}
	return google.DefaultClient(ctx, computeInstancesGetScope)
}

func lastPathSegment(s string) string {
	s = strings.TrimRight(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
