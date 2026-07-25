// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package railway provides a [resource.Detector] which supports detecting
attributes specific to services running on Railway (https://railway.com).

Attributes are derived from Railway's provider-injected environment
variables (https://docs.railway.com/variables/reference). The detector
reports an empty resource, without error, when RAILWAY_PROJECT_ID is unset,
since that indicates the process is not running on Railway.

According to semantic conventions for [cloud], [service], [deployment], and
[vcs] attributes, each of the following attributes is added if the
corresponding environment variable is set:

  - cloud.provider (always "railway" - not one of the well-known values in
    the cloud.provider registry, but the specification explicitly allows a
    custom value when the provider isn't listed)
  - cloud.region (from RAILWAY_REPLICA_REGION)
  - service.namespace (from RAILWAY_PROJECT_NAME)
  - service.name (from RAILWAY_SERVICE_NAME)
  - service.instance.id (from RAILWAY_REPLICA_ID)
  - deployment.environment.name (from RAILWAY_ENVIRONMENT_NAME)
  - deployment.id (from RAILWAY_DEPLOYMENT_ID)
  - vcs.ref.head.revision (from RAILWAY_GIT_COMMIT_SHA)
  - vcs.ref.head.name (from RAILWAY_GIT_BRANCH)
  - vcs.ref.head.type (always "branch" when RAILWAY_GIT_BRANCH is set, since
    Railway only deploys from branches, never tags)
  - vcs.repository.name (from RAILWAY_GIT_REPO_NAME)
  - vcs.owner.name (from RAILWAY_GIT_REPO_OWNER)
  - vcs.provider.name (always "github" when RAILWAY_GIT_REPO_OWNER is set,
    since Railway's git integration is GitHub-only today)

RAILWAY_PROJECT_ID, RAILWAY_ENVIRONMENT_ID, and RAILWAY_SERVICE_ID have no
generic semantic convention equivalent, so they are reported under a
package-specific namespace instead:

  - railway.project.id (from RAILWAY_PROJECT_ID)
  - railway.environment.id (from RAILWAY_ENVIRONMENT_ID)
  - railway.service.id (from RAILWAY_SERVICE_ID)

The following Railway environment variables are intentionally not mapped to
resource attributes:

  - RAILWAY_PUBLIC_DOMAIN, RAILWAY_PRIVATE_DOMAIN, RAILWAY_TCP_PROXY_DOMAIN,
    RAILWAY_TCP_PROXY_PORT, RAILWAY_TCP_APPLICATION_PORT: networking
    configuration, not resource identity.
  - RAILWAY_VOLUME_NAME, RAILWAY_VOLUME_MOUNT_PATH: storage configuration,
    only present when a volume is attached.
  - RAILWAY_SNAPSHOT_ID: an internal rollback pointer with limited
    observability value.
  - RAILWAY_GIT_COMMIT_MESSAGE, RAILWAY_GIT_AUTHOR: free-form text with no
    corresponding semantic convention; resource attributes are expected to
    be low-cardinality and stable.

The vcs.* attributes are still Release Candidate stability in the semantic
conventions registry, not yet Stable. They've been renamed before (e.g.
vcs.repository.ref.name became vcs.ref.head.name) and could be renamed
again in a future semantic-conventions release.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/README.md
[deployment]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/deployment-environment.md
[vcs]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/README.md
*/
package railway // import "go.opentelemetry.io/contrib/detectors/railway"
