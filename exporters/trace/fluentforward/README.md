### Opentelemetry exporter for fluentbit

WIP, the demo application can be found [here](https://github.com/Syn3rman/community-bridge/tree/master/ffexample)

This module contains an exporter that exports span data from the opentelemetry go sdk to a fluentbit/fluentd instance using the fluentforward protocol.

The exporter can be installed in your application using the `InstallNewPipeline` function.

```
err := fluentforward.InstallNewPipeline("localhost:24224", "fluentforward")
```

The document that maps opentelemetry types to fluent-compatible type can be found [here](https://docs.google.com/document/d/1N1cqaLnnl-lmwyIbgJvbdGacXhxBTG7ywxWz6qTbUMY/edit?usp=sharing)