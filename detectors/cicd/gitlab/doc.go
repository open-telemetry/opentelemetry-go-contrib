// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package gitlab provides a [resource.Detector] which supports detecting
attributes specific to Gitlab CI.

According to semantic conventions for [cicd] and [vcs] attributes,
each of the following attributes is added if it is available:

  - cicd.pipeline.name
  - cicd.pipeline.task.run.id
  - cicd.pipeline.task.name
  - cicd.pipeline.task.type
  - cicd.pipeline.run.id
  - cicd.pipeline.task.run.url.full
  - vcs.repository.ref.name
  - vcs.repository.ref.type
  - vcs.repository.change.id
  - vcs.repository.url.full

[cicd]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/attributes-registry/cicd.md
[vcs]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/attributes-registry/vcs.md
*/
package gitlab // import "go.opentelemetry.io/contrib/detectors/cicd/gitlab"
