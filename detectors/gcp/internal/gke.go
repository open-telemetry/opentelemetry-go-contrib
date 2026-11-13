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
	"errors"
	"fmt"
	"strings"
)

const (
	// If the kubernetes.default.svc service exists in the cluster,
	// then the KUBERNETES_SERVICE_HOST env var will be populated.
	// Use this as an indication that we are running on kubernetes.
	k8sServiceHostEnv = "KUBERNETES_SERVICE_HOST"
	// See the available GKE metadata:
	// https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity#instance_metadata
	clusterNameMetadataAttr     = "cluster-name"
	clusterLocationMetadataAttr = "cluster-location"
)

func (d *Detector) onGKE(ctx context.Context) bool {
	// Check if we are on k8s first
	_, found := d.os.LookupEnv(k8sServiceHostEnv)
	if !found {
		return false
	}
	// If we are on k8s, make sure that we are actually on GKE, and not a
	// different managed k8s platform.
	_, err := d.metadata.InstanceAttributeValueWithContext(ctx, clusterLocationMetadataAttr)
	return err == nil
}

// GKEHostID returns the instance ID of the instance on which this program is running.
func (d *Detector) GKEHostID(ctx context.Context) (string, error) {
	return d.GCEHostID(ctx)
}

// GKEClusterName returns the name of the GKE cluster in which this program is running.
func (d *Detector) GKEClusterName(ctx context.Context) (string, error) {
	return d.metadata.InstanceAttributeValueWithContext(ctx, clusterNameMetadataAttr)
}

// GKEHostType returns the machine type of the instance on which this program is running.
func (d *Detector) GKEHostType(ctx context.Context) (string, error) {
	instance, err := d.computeInstance(ctx)
	if err != nil {
		return "", err
	}
	if instance.MachineType == "" {
		return "", errors.New("compute instances.get response missing machineType")
	}
	return lastPathSegment(instance.MachineType), nil
}

// LocationType represents the type of location (zone or region).
type LocationType int64

// Supported location types.
const (
	UndefinedLocation LocationType = iota
	Zone
	Region
)

// GKEAvailabilityZoneOrRegion returns the location of the cluster and whether the cluster is zonal or regional.
func (d *Detector) GKEAvailabilityZoneOrRegion(ctx context.Context) (string, LocationType, error) {
	clusterLocation, err := d.metadata.InstanceAttributeValueWithContext(ctx, clusterLocationMetadataAttr)
	if err != nil {
		return "", UndefinedLocation, err
	}
	switch strings.Count(clusterLocation, "-") {
	case 1:
		return clusterLocation, Region, nil
	case 2:
		return clusterLocation, Zone, nil
	default:
		return "", UndefinedLocation, fmt.Errorf("unrecognized format for cluster location: %v", clusterLocation)
	}
}
