// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/contrib/detectors/service"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type serviceDetector struct {
}

// New returns a [resource.Detector] that will detect service resources.
func New() resource.Detector {
	return &serviceDetector{}
}

// Detect detects resources associated to a service.
func (detector *serviceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	version4Uuid, err := uuid.NewRandom()

	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceInstanceID(version4Uuid.String()),
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}
