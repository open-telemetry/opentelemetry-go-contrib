### Opentelemetry exporter for fluentbit

WIP, the demo application can be found [here](https://github.com/Syn3rman/community-bridge/tree/master/ffexample)

This module contains an exporter that exports span data from the opentelemetry go sdk to a fluentbit/fluentd instance using the fluentforward protocol.


### Installation

The exporter can be installed in your application using the `InstallNewPipeline` function.

```
err := fluentforward.InstallNewPipeline("localhost:24224", "fluentforward")
```

To set up the pipeline, `InstallNewPipeline()` calls `NewExportPipeline()`, which in turn calls `NewRawExporter()`.

`InstallNewPipeline()` instantiates a NewExportPipeline with the recommended configuration and registers it globally. `NewExportPipeline()` sets up the export pipeline with the recommended configuration for the tracer provider, and `NewRawExporter()` creates the exporter.

### Data mapping

The mapping for a span is given below:

| Otel-go representation   | Otel type          | Fluentbit representation   | Fluentbit type            |
| ------------------------ | -----------        | -------------------------- | ----------------          |
| TraceID                  | trace.ID           | TraceID                    | string                    |
| SpanID                   | trace.SpanID       | SpanID                     | string                    |
| ParentSpanID             | trace.SpanID       | ParentSpanID               | string                    |
| Name                     | string             | Name                       | string                    |
| SpanKind                 | trace.SpanKind     | SpanKind                   | int                       |
| StartTime                | time.Time          | StartTime                  | int64                     |
| EndTime                  | time.Time          | EndTime                    | int64                     |
| Attributes               | []label.KeyValue   | Attrs                      | map[label.Key]interface{} |
| DroppedAttributesCount   | int                | DroppedAttributeCount      | int                       |
| MessageEvents            | trace.Event        | MessageEvents              | []event                   |
| DroppedEventsCount       | int                | DroppedEventsCount         | int                       |
| Links                    | []trace.Link       | Links                      | []link                    |
| DroppedLinksCount        | int                | DroppedLinksCount          | int                       |
| StatusCode               | codes.Code         | StatusCode                 | string                    |
| StatusMessage            | string             | StatusMessage              | string                    |
| Resource                 | *resource.Resource | Resource                   | string                    |
| InstrumentationLibrary   | Instrumentation.Library | InstrumentationLibraryName, Version | string |


 Note: The field TraceState is missing in the go implementation

The mappings for an event and a link are as shown:

#### Link


| Otel-go representation   | Otel type        | Fluentbit representation   | Fluentbit type            |
| ------------------------ | -----------      | -------------------------- | ----------------          |
| TraceID                  | trace.ID         | TraceID                    | string                    |
| SpanID                   | trace.SpanID     | SpanID                     | string                    |
| Attributes               | []label.KeyValue | Attrs                      | map[label.Key]interface{} |


#### Event


| Otel-go representation   | Otel type        | Fluentbit representation   | Fluentbit type            |
| ------------------------ | -----------      | -------------------------- | ----------------          |
| Name                     | string           | Name                       | string                    |
| Time                     | time.Time        | Time                       | int64                     |
| Attributes               | []label.KeyValue | Attrs                      | map[label.Key]interface{} |
