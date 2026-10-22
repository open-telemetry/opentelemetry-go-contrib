// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import "maps"

func mergeAdditionalProperties(
	existing any,
	raw map[string]any,
) any {
	if len(raw) == 0 {
		return existing
	}

	switch properties := existing.(type) {
	case map[string]any:
		merged := make(map[string]any, len(properties)+len(raw))
		maps.Copy(merged, properties)
		maps.Copy(merged, raw)
		return merged

	case map[any]any:
		merged := make(map[any]any, len(properties)+len(raw))
		maps.Copy(merged, properties)
		for key, value := range raw {
			merged[key] = value
		}
		return merged

	default:
		merged := make(map[string]any, len(raw))
		maps.Copy(merged, raw)
		return merged
	}
}

func mergeExplicitAdditionalProperties(
	explicit any,
	raw map[string]any,
) any {
	switch properties := explicit.(type) {
	case map[string]any:
		merged := make(map[string]any, len(raw)+len(properties))
		maps.Copy(merged, raw)
		maps.Copy(merged, properties)
		return merged

	case map[any]any:
		merged := make(map[any]any, len(raw)+len(properties))
		for key, value := range raw {
			merged[key] = value
		}
		maps.Copy(merged, properties)
		return merged

	default:
		if len(raw) == 0 {
			return explicit
		}

		merged := make(map[string]any, len(raw))
		maps.Copy(merged, raw)
		return merged
	}
}
