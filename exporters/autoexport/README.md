# Automatic Exporter configuration

This module provides easy access to configuring a trace exporter that can be used when configuring an OpenTelemetry Go SDK trace export pipeline.

The autoexport package looks for the `OTEL_TRACES_EXPORTER` environment variable and if set, attempts to load the exporter from it's registry of exporters. The registry is always loaded with an OTLP exporter with the key `"otlp"` and additional exporters can be registered using `autoexport.RegisterSpanExporter("name", ...)`(See example below). Exporter registration uses a factory pattern to not unneccarily build exporters and use resources until they are requested.

If the environment variable is not set, the fallback exporter is returned. The fallback exporter defaults to an [OTLP exporter](https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace) and can be overriden using the `WithFallbackSpanExporter` option.

### Getting an exporter

Set the preferred exporter type using the `OTEL_TRACES_EXPORTER` environment variable.

```shell
export OTEL_TRACES_EXPORTER="otlp"
```

```golang
import "go.opentelemetry.io/contrib/exporters/autoexport"
...

exp, err := autoexport.NewTraceExporter(context.Background())
```

### Defining a custom fallback exporter

The fallback exporter is returned if the exporter environment variable is not set.

```golang
import "go.opentelemetry.io/contrib/exporters/autoexport"

fallbackExp = ...
exp, err := autoexport.NewTraceExporter(
    context.Background(),
    WithFallbackSpanExporter(fallbackExp),
)
```

### Registering a custom exporter

Set the `OTEL_TRACES_EXPORTER` environment variable to your custom exporter's name.

```shell
export OTEL_TRACES_EXPORTER="my-custom-exporter"
```

Register your custom exporter's factory before retrieveing it.

```golang
import "go.opentelemetry.io/contrib/exporters/autoexport"

autoexport.RegisterSpanExporter("my-custom-exporter", func(ctx context.Context) (trace.SpanExporter, error) {
    exp := ...
    return exp, nil
})
exp, err := autoexport.NewTraceExporter(context.Background())
```
