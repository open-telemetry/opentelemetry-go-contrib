# How to usage

This library is the instrumentation library for `google.golang.org/grpc`

For now you can instrument your program which use `google.golang.org/grpc` in two ways:

- by gRPC Interceptors
- [experimental] by gPRC `stats.Handler`

You can see the example of both ways in directory `./example`

Although the implementation `stats.Handler` in experimental stage, we strongly still recommand you to use `stats.Handler`, mainly for two reasons:
- **Functional advantages**: `stats.Handler`` has more information for user to build more flexible and granular metric, for example
  - multiple different types of represent "data length": In [InPayLoad](https://pkg.go.dev/google.golang.org/grpc/stats#InPayload), there exists `Length`, `CompressedLength`, `WireLength` to denote the size of uncompressed, compressed payload data, with or without framing data. But in Interceptors, we can only got uncompressed data, and this feature is also removed due to performance problem. [#3168](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/3168)
  - more accurate timestamp: `InPayload.RecvTime` and `OutPayload.SentTime` records more accurate timestamp that server got and sent the message, the timestamp recorded by interceptors depends on the location of this interceptors in the total interceptor chain.
  - some other use cases: for example [catch failure of decoding message](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/197#issuecomment-668377700)
- **Performance advantages**: If too many interceptors are registered in a service, the interceptor chain can become too long, which increases the latency and processing time of the entire RPC call.

You should also **notice** that: **Do not use both two ways in the meantime!** If so, you will get duplicated spans and the parent/child relationships between spans will also be broken.
