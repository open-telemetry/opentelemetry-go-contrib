# gRPC Tracing Example

Traces client and server calls using the OpenTelemetry gRPC stats handlers.

Run the commands below from this example directory. Start the server first;
then run the client in a second terminal. Both write trace data to standard output.
Stop the server with Ctrl+C.

## Compile .proto

Only required if the service definition (.proto) changes.

```sh
# protobuf v34.0
protoc -I api --go_out=paths=source_relative:./api --go-grpc_out=paths=source_relative:./api api/hello-service.proto
```

## Run server

```sh
go run ./server
```

## Run client

```sh
go run ./client
```
