// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azurecontainerapps provides a resource detector which supports
detecting attributes specific to Azure Container Apps.

Note: Azure Container Apps jobs are not supported.

According to semantic conventions for [cloud] and [service] attributes,
the following attributes are added when running on Azure Container Apps:

  - cloud.provider
  - cloud.platform
  - service.name
  - service.instance.id

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
*/
package azurecontainerapps // import "go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
