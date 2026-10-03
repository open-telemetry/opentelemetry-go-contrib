// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"

	"go.opentelemetry.io/contrib/detectors/docker/internal"
)

func getContainerID(context.Context) (string, error) {
	return internal.ContainerID()
}
