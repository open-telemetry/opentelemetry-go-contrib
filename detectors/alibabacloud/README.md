# Alibaba Cloud Resource Detector for Go

<!--[![Go Reference][goref-image]][goref-url]-->
<!--[goref-image]: https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/alibabacloud.svg-->
<!--[goref-url]: https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/alibabacloud-->

The Alibaba Cloud resource detector supports detecting resources on:

* Elastic Compute Service (ECS) instances
* Elastic Container Instances (ECI), including pods on Alibaba Cloud
  Container Compute Service (ACS)
* Pods on Alibaba Cloud Container Service for Kubernetes (ACK) clusters
  running on ECS nodes, where it reports the metadata of the underlying
  ECS node

Both environments expose the same instance metadata service at
`100.100.100.200`. On an ECI pod the `hostname` item is the pod name and the
`instance-id` item is an `eci-` prefixed identifier. Because semantic
conventions define no cloud platform value for ECI, containers report the
same `alibaba_cloud_ecs` `cloud.platform` value as ECS instances.

## Usage

```golang
ctx := context.Background()
// Detect your resources
res, err := resource.New(ctx,
    // Use the Alibaba Cloud resource detector!
    resource.WithDetectors(alibabacloud.NewResourceDetector()),
    // Keep the default detectors
    resource.WithTelemetrySDK(),
    // Add your own custom attributes to identify your application
    resource.WithAttributes(
        semconv.ServiceNameKey.String("my-application"),
        semconv.ServiceNamespaceKey.String("my-company-frontend-team"),
    ),
)
if err != nil {
    // Handle err
}
// Use the resource in your TracerProvider (or MeterProvider)
tp := trace.NewTracerProvider(
    // ... other options
    trace.WithResource(res),
)
```

## Detected attributes

The detector always adds `cloud.provider` (`alibaba_cloud`) and
`cloud.platform` (`alibaba_cloud_ecs`) once it confirms the process is
running on Alibaba Cloud. Each of the following attributes is then added if
the corresponding metadata item is available:

| Metadata item              | Attribute                                        |
|----------------------------|--------------------------------------------------|
| `instance-id`              | `cloud.resource_id`, `host.id`                   |
| `region-id`                | `cloud.region`                                   |
| `zone-id`                  | `cloud.availability_zone`                        |
| `owner-account-id`         | `cloud.account.id`                               |
| `hostname`                 | `host.name`                                      |
| `instance/instance-type`   | `host.type`                                      |
| `image-id`                 | `host.image.id`                                  |
| `mac`                      | `host.mac`                                       |
| `private-ipv4`             | `host.ip`                                        |
| `public-ipv4`              | `host.ip` (appended when not already present)    |
| `eipv4`                    | `host.ip` (appended when not already present)    |
| `vpc-id`                   | `alibaba.cloud.vpc.id`                           |
| `vswitch-id`               | `alibaba.cloud.vswitch.id`                       |

The `alibaba.cloud.vpc.id` and `alibaba.cloud.vswitch.id` identifiers use a
vendor namespace because semantic conventions do not (yet) define
counterparts for them.

## Metadata service access

The detector prefers the hardened [IMDSv2](https://www.alibabacloud.com/help/en/ecs/user-guide/manage-instance-metadata)
flow: it first requests a session token via `PUT /latest/api/token` and
sends that token with every metadata request. When the token endpoint is
unavailable, the detector falls back to unauthenticated IMDSv1 requests.
Both hardened instances (where IMDSv1 is rejected) and environments without
token support, such as ECI, therefore work with the same detector.

All requests use a two second timeout, a direct transport without proxy
(the metadata endpoint is a link-local address that must not be reached via
`HTTP_PROXY`/`HTTPS_PROXY`), and responses are capped in size.

## Behavior on error

The detection outcome depends on how the metadata service responds:

* Not running on Alibaba Cloud (metadata service unreachable or the
  `instance-id` item is absent): an empty resource is returned with no
  error, so the detector can safely be combined with detectors for other
  providers.
* Metadata service reachable but failing (e.g. a 5xx response for
  `instance-id`): an error is returned.
* Running on Alibaba Cloud, but some optional items are unavailable: the
  attributes that could be retrieved are returned, together with
  `resource.ErrPartialResource`.

## Kubernetes clusters

The metadata service is reachable from pods on ACK clusters, and the
detector works there with both the IMDSv2 and IMDSv1 flows. What it reports
depends on where the pod runs:

* **Regular ACK node pools (ECS nodes)**: the metadata service answers with
  the node's metadata. `host.id` and `host.name` identify the underlying ECS
  instance, not the pod.
* **Serverless node pools (ECI, ACS)**: the metadata service answers with
  pod-level metadata. The `hostname` item is the pod name and the
  `instance-id` item is an `eci-` prefixed identifier.

If the cluster restricts pod access to the metadata service (e.g. via ACK
metadata access control), the detector reports an empty resource with no
error, the same as when not running on Alibaba Cloud.

Pod-level identity (`k8s.pod.name`, `k8s.namespace.name`) is not exposed by
the metadata service. For full Kubernetes attributes use the
[detectors/k8sapi](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/k8sapi)
detector in addition to this one.
