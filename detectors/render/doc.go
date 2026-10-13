// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package render provides a [resource.Detector] which detects attributes of
// services running on Render (https://render.com).
//
// The following attributes are added from the environment variables that
// Render sets on every service:
//
//   - service.instance.id (RENDER_INSTANCE_ID, or RENDER_SERVICE_ID if unset)
//   - service.name (RENDER_SERVICE_NAME, or RENDER_SERVICE_ID if unset)
//   - service.version (RENDER_GIT_COMMIT)
//
// When RENDER_SERVICE_ID is not set, an empty resource is returned.
package render
