// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package elasticbeanstalk provides a resource detector which supports
detecting attributes specific to AWS Elastic Beanstalk

According to semantic conventions for [cloud], [deployment], and [service] attributes,
the following attributes are added when running on AWS Elastic Beanstalk:

  - cloud.provider
  - cloud.platform
  - deployment.environment.name
  - service.instance.id
  - service.version

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[deployment]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/deployment-environment.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
*/
package elasticbeanstalk // import "go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"
