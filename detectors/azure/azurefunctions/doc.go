// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azurefunctions provides a resource detector which supports
detecting attributes specific to Azure Functions.

The detector gates on FUNCTIONS_WORKER_RUNTIME or
FUNCTIONS_EXTENSION_VERSION, since Azure Functions shares most WEBSITE_*
environment variables with Azure App Service.

According to semantic conventions for [cloud], [service], and [faas]
attributes, and a custom attribute for the Azure resource group, the
following attributes are added when running on Azure Functions:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.account.id
  - cloud.resource_id
  - service.name
  - azure.resource_group.name
  - faas.instance
  - deployment.environment.name

cloud.resource_id, azure.resource_group.name, and faas.instance are
sourced defensively and vary by hosting plan, so they may be absent from
the resulting resource.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
[faas]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/faas.md
*/
package azurefunctions // import "go.opentelemetry.io/contrib/detectors/azure/azurefunctions"
