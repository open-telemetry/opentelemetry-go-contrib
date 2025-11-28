// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/autodetect"
	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func toDetectorId(detectors []ExperimentalResourceDetector) []autodetect.ID {
	ids := []autodetect.ID{}
	for _, d := range detectors {
		if d.Container != nil {
			ids = append(ids, autodetect.IDContainer)
		}
		if d.Host != nil {
			ids = append(ids, autodetect.IDHost, autodetect.IDHostID)
		}
		if d.Process != nil {
			ids = append(ids,
				autodetect.IDProcessCommandArgs,
				autodetect.IDProcessExecutableName,
				autodetect.IDProcessExecutablePath,
				autodetect.IDProcessOwner,
				autodetect.IDProcessPID,
				autodetect.IDProcessRuntimeDescription,
				autodetect.IDProcessRuntimeName,
				autodetect.IDProcessRuntimeVersion,
			)
		}
	}
	return ids
}

func newResource(res OpenTelemetryConfigurationResource) (*resource.Resource, error) {
	if res == nil {
		return resource.Default(), nil
	}

	r, ok := res.(*ResourceJson)
	if !ok {
		return nil, newErrInvalid("resource")
	}

	attrs := make([]attribute.KeyValue, 0, len(r.Attributes))
	for _, v := range r.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	var detectedResource *resource.Resource

	if r.DetectionDevelopment != nil {
		detectors := toDetectorId(r.DetectionDevelopment.Detectors)
		fmt.Println(detectors)
		detector, err := autodetect.Detector(detectors...)
		if err != nil {
			return nil, err
		}
		detectedResource, err = detector.Detect(context.Background())
		if err != nil {
			return nil, err
		}
	}

	if r.SchemaUrl == nil {
		return resource.Merge(resource.NewSchemaless(attrs...), detectedResource)
	}
	return resource.Merge(resource.NewWithAttributes(*r.SchemaUrl, attrs...), detectedResource)
}
