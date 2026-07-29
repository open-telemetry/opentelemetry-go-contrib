// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package elasticbeanstalk provides a resource detector for AWS Elastic Beanstalk.

It reads the AWS X-Ray daemon configuration file written by the Elastic Beanstalk
platform. X-Ray integration must be enabled on the environment for this file to
be present, without it the detector returns an empty resource silently.

Configuration file locations:

  - Linux:   /var/elasticbeanstalk/xray/environment.conf
  - Windows: C:\Program Files\Amazon\XRay\environment.conf

The following attributes are populated according to semantic conventions for
[cloud], [deployment], and [service] resources:

  - cloud.provider          (e.g. "aws")
  - cloud.platform          (e.g. "aws_elastic_beanstalk")
  - deployment.environment.name (e.g. "production")
  - deployment.id           (e.g. "23")
  - service.version         (e.g. "env-version-1234")

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[deployment]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/deployment-environment.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md

[X-Ray integration must be enabled]: https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/environment-configuration-debugging.html
*/
package elasticbeanstalk
