// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azureappservice provides a resource detector which supports
detecting attributes specific to Azure App Service.

The detector gates on WEBSITE_SITE_NAME, WEBSITE_RESOURCE_GROUP, and
WEBSITE_OWNER_NAME, and defers to the Functions detector when
FUNCTIONS_WORKER_RUNTIME is also set, since Azure Functions runs on the
same App Service infrastructure.

According to semantic conventions for [cloud], [service], and [deployment]
attributes, and a custom attribute for the Azure resource group and
instance id, the following attributes are added when running on Azure App
Service:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.account.id
  - cloud.resource_id
  - service.name
  - azure.resource_group.name
  - azure.app_service.instance.id
  - deployment.environment.name

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
[deployment]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/deployment-environment.md
*/
package azureappservice
