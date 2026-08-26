// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2024 Google LLC
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

// This provides an always-passing test that simply logs the detected environment in which it is run.
// To run it on an arbitrary machine, build a test binary:
//
// go test -c -tags print
//
// and then copy it to a test VM and run:
//
// ./gcp.test -test.v -test.run TestDetectorPrint

//go:build print

package internal

import "testing"

func TestDetectorPrint(t *testing.T) {
	d := NewDetector()
	t.Logf("CloudPlatform: %+v", d.CloudPlatform())
	log := func(label, v interface{}, err error) {
		t.Helper()
		if err != nil {
			t.Logf("%s returned err %v", label, err)
		} else {
			t.Logf("%s = %+v", label, v)
		}
	}
	v, err := d.ProjectID()
	log("ProjectID", v, err)
	v, err = d.AppEngineServiceInstance()
	log("AppEngineServiceInstance", v, err)
	v, err = d.AppEngineServiceName()
	log("AppEngineServiceName", v, err)
	v, err = d.AppEngineServiceVersion()
	log("AppEngineServiceVersion", v, err)
	v, err = d.AppEngineStandardAvailabilityZone()
	log("AppEngineStandardAvailabilityZone", v, err)
	v, err = d.AppEngineStandardCloudRegion()
	log("AppEngineStandardCloudRegion", v, err)
	v1, v2, err := d.AppEngineFlexAvailabilityZoneAndRegion()
	log("AppEngineFlexAvailabilityZoneAndRegion.Zone", v1, err)
	log("AppEngineFlexAvailabilityZoneAndRegion.Region", v2, err)
	v, err = d.BareMetalSolutionCloudRegion()
	log("BareMetalSolutionCloudRegion", v, err)
	v, err = d.BareMetalSolutionInstanceID()
	log("BareMetalSolutionInstanceID", v, err)
	v, err = d.BareMetalSolutionProjectID()
	log("BareMetalSolutionProjectID", v, err)
	v, err = d.CloudRunJobExecution()
	log("CloudRunJobExecution", v, err)
	v, err = d.CloudRunJobTaskIndex()
	log("CloudRunJobTaskIndex", v, err)
	v, err = d.FaaSCloudRegion()
	log("FaaSCloudRegion", v, err)
	v, err = d.FaaSID()
	log("FaaSID", v, err)
	v, err = d.FaaSName()
	log("FaaSName", v, err)
	v, err = d.FaaSVersion()
	log("FaaSVersion", v, err)
	v1, v2, err = d.GCEAvailabilityZoneAndRegion()
	log("GCEAvailabilityZoneAndRegion.Zone", v1, err)
	log("GCEAvailabilityZoneAndRegion.Region", v1, err)
	v, err = d.GCEHostID()
	log("GCEHostID", v, err)
	v, err = d.GCEHostName()
	log("GCEHostName", v, err)
	v, err = d.GCEHostType()
	log("GCEHostType", v, err)
	v, err = d.GCEInstanceHostname()
	log("GCEInstanceHostname", v, err)
	v, err = d.GCEInstanceName()
	log("GCEInstanceName", v, err)
	mig, err := d.GCEManagedInstanceGroup()
	log("GCEManagedInstanceGroup", mig, err)
	loc, locType, err := d.GKEAvailabilityZoneOrRegion()
	log("GKEAvailabilityZoneOrRegion.Location", loc, err)
	log("GKEAvailabilityZoneOrRegion.Type", locType, err)
	v, err = d.GKEClusterName()
	log("GKEClusterName", v, err)
	v, err = d.GKEHostID()
	log("GKEHostID", v, err)
}
