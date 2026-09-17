# GCP Resource detector

The GCP resource detector supports detecting resources on:

* Google Compute Engine (GCE)
* Google Kubernetes Engine (GKE)
* Google App Engine (GAE)
* Cloud Run
* Cloud Run jobs
* Cloud Run worker pools
* Cloud Functions
* Bare Metal Solution (BMS)

## Usage

```golang
ctx := context.Background()
// Detect your resources
res, err := resource.New(ctx,
    // Use the GCP resource detector!
    resource.WithDetectors(gcp.NewDetector()),
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
// Use the resource in your tracerprovider (or meterprovider)
tp := trace.NewTracerProvider(
    // ... other options
    trace.WithResource(res),
)
```

## Setting Kubernetes attributes

Previous iterations of GCP resource detection attempted to detect
`container.name`, `k8s.pod.name` and `k8s.namespace.name`.  When using this detector,
you should use this in your Pod Spec to set these using
[`OTEL_RESOURCE_ATTRIBUTES`](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.20.0/specification/resource/sdk.md#specifying-resource-information-via-an-environment-variable):

```yaml
env:
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: NAMESPACE_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: CONTAINER_NAME
  value: my-container-name
- name: OTEL_RESOURCE_ATTRIBUTES
  value: k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME)
```

To have a detector unpack the `OTEL_RESOURCE_ATTRIBUTES` envvar, use the `WithFromEnv` option:

```golang
...
// Detect your resources
res, err := resource.New(ctx,
    resource.WithDetectors(gcp.NewDetector()),
    resource.WithTelemetrySDK(),
    resource.WithFromEnv(), // unpacks OTEL_RESOURCE_ATTRIBUTES
    // Add your own custom attributes to identify your application
    resource.WithAttributes(
        semconv.ServiceNameKey.String("my-application"),
        semconv.ServiceNamespaceKey.String("my-company-frontend-team"),
    ),
)
...
```

## Bare Metal Solution (BMS)

Bare Metal Solution hosts do not provide a Compute Engine metadata server. Detection on BMS is entirely environment-driven and requires all three of the following environment variables to be set and non-empty (matching the [Google Cloud Ops Agent BMS configuration](https://docs.cloud.google.com/bare-metal/docs/set-up-ops-agent-for-bms#authorize_the_ops_agent)):

* `BMS_PROJECT_ID`: Mapped to `cloud.account.id`.
* `BMS_LOCATION` (or `BMS_REGION` alias): Mapped to `cloud.region`.
* `BMS_INSTANCE_ID`: Mapped to `host.id`.

When detected, `cloud.provider` is set to `gcp` and `cloud.platform` is set to `gcp_bare_metal_solution`. Without all three non-empty environment variables configured, no BMS resource is detected.
