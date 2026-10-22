// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import "maps"

func mergeAdditionalProperties(
	existing any,
	raw map[string]any,
) any {
	properties, _ := existing.(map[string]any)

	if len(raw) == 0 {
		return existing
	}

	merged := make(map[string]any, len(properties)+len(raw))
	maps.Copy(merged, properties)

	maps.Copy(merged, raw)

	return merged
}

func mergeExplicitAdditionalProperties(
	explicit any,
	raw map[string]any,
) any {
	properties, _ := explicit.(map[string]any)

	merged := make(map[string]any, len(properties)+len(raw))

	maps.Copy(merged, raw)

	maps.Copy(merged, properties)

	if len(merged) == 0 {
		return explicit
	}

	return merged
}
