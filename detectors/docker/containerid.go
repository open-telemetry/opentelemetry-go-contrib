// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"

	"go.opentelemetry.io/contrib/detectors/docker/internal"
)

type containerIDProvider func(context.Context) (string, error)

func getContainerID(context.Context) (string, error) {
	return internal.ContainerID()
}
