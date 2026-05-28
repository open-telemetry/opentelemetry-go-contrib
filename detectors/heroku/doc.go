// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package heroku provides a [resource.Detector] which supports detecting
attributes specific to Heroku dynos.

According to semantic conventions for [cloud], [heroku], and [service]
attributes, each of the following attributes is added if it is available:

  - cloud.provider
  - heroku.app.id
  - heroku.release.commit
  - heroku.release.creation_timestamp
  - service.instance.id
  - service.name
  - service.version

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[heroku]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/heroku.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
*/
package heroku // import "go.opentelemetry.io/contrib/detectors/heroku"
