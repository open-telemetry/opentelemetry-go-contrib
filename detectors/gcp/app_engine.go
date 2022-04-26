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

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func onAppEngine() bool {
	return os.Getenv("GAE_SERVICE") != ""
}

func appEngineAttributes(ctx context.Context) (attributes []attribute.KeyValue, errs []string) {
	// Part of GAE runtime contract.
	// See https://cloud.google.com/appengine/docs/flexible/python/migrating#modules
	if serviceName := os.Getenv("GAE_SERVICE"); serviceName == "" {
		errs = append(errs, "envvar GAE_SERVICE contains empty string.")
	} else {
		attributes = append(attributes, semconv.FaaSNameKey.String(serviceName))
	}
	if serviceVersion := os.Getenv("GAE_VERSION"); serviceVersion == "" {
		errs = append(errs, "envvar GAE_VERSION contains empty string.")
	} else {
		attributes = append(attributes, semconv.FaaSVersionKey.String(serviceVersion))
	}
	if serviceInstance := os.Getenv("GAE_INSTANCE"); serviceInstance == "" {
		errs = append(errs, "envvar GAE_INSTANCE contains empty string.")
	} else {
		attributes = append(attributes, semconv.FaaSIDKey.String(serviceInstance))
	}

	// TODO region?
	return
}
