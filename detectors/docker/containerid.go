// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type containerIDProvider func(context.Context) (string, error)

func getContainerID(ctx context.Context) (string, error) {
	res, err := resource.New(ctx, resource.WithContainerID())
	if err != nil {
		return "", err
	}

	id, ok := res.Set().Value(semconv.ContainerIDKey)
	if !ok {
		return "", nil
	}
	return id.AsString(), nil
}
