### Opentelemetry exporter for fluentbit

WIP, the demo application can be found [here](https://github.com/Syn3rman/community-bridge/tree/master/ffexample)

This module contains an exporter that exports span data from the opentelemetry go sdk to a FluentBit/Fluentd instance using the [Fluentd Forward protocol](https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1).


### Installation

Provided that you have a FluentBit instance running on the node, you can export spans to the collector (that has the fluentforwardreceiver enabed) or any other fluent instance that is listening using the fluentforward protocol by installing this exporter in your application by simply using the InstallNewPipeline() method as shown:

```
hostport := "localhost:24224"
serviveName := "fluentforward"
retryTimeout := 10
err := fluentforward.InstallNewPipeline(hostport, serviceName, retryTimeout)
```

To set up the pipeline, `InstallNewPipeline()` calls `NewExportPipeline()`, which in turn calls `NewRawExporter()`.

`InstallNewPipeline()` instantiates a NewExportPipeline with the recommended configuration and registers it globally. `NewExportPipeline()` sets up the export pipeline with the recommended configuration for the tracer provider, and `NewRawExporter()` creates the exporter.

While initializing the exporter, we can specify the timeout parameter, which will reconnect to the remote instance every timeout duration, which decreases the chances of writes failing due to connection issues.

If a write fails, the exporter will attempt to reconnect to the fluent instance and will retry the write again before returning.
### Data mapping

The data mapping can be found below:

For a span:

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
| Resource                 | *resource.Resource | Resource                   | map[label.Key]interface{} |
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


> [To do]: A blog post detailing the installation of the exporter in your application and configuring the FluentBit instance with the collector can be found here (add link after the blog is published)