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
	"strings"

	"cloud.google.com/go/compute/metadata"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func onCloudRun() bool {
	return os.Getenv("K_CONFIGURATION") != ""
}

func onCloudFunctions() bool {
	return os.Getenv("FUNCTION_TARGET") != ""
}

func faasAttributes(ctx context.Context) (attributes []attribute.KeyValue, errs []string) {
	// Part of Cloud Run and Cloud Functions container runtime contracts.
	// See https://cloud.google.com/run/docs/reference/container-contract and
	// https://cloud.google.com/functions/docs/configuring/env-var#runtime_environment_variables_set_automatically
	if serviceName := os.Getenv("K_SERVICE"); serviceName == "" {
		errs = append(errs, "envvar K_SERVICE contains empty string.")
	} else {
		attributes = append(attributes, semconv.FaaSNameKey.String(serviceName))
	}
	if serviceVersion := os.Getenv("K_REVISION"); serviceVersion == "" {
		errs = append(errs, "envvar K_REVISION contains empty string.")
	} else {
		attributes = append(attributes, semconv.FaaSVersionKey.String(serviceVersion))
	}
	if instanceID, err := metadata.InstanceID(); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if instanceID != "" {
		attributes = append(attributes, semconv.FaaSIDKey.String(instanceID))
	}
	if region, err := faasCloudRegion(ctx); hasProblem(err) {
		errs = append(errs, err.Error())
	} else if region != "" {
		attributes = append(attributes, semconv.CloudRegionKey.String(region))
	}
	return
}

func faasCloudRegion(ctx context.Context) (string, error) {
	region, err := metadata.Get("instance/region")
	if err != nil {
		return "", err
	}
	// Region from the metadata server is in the format /projects/123/regions/r.
	// https://cloud.google.com/run/docs/reference/container-contract#metadata-server
	return region[strings.LastIndex(region, "/")+1:], nil
}
