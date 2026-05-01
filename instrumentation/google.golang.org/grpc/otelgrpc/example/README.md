# gRPC Tracing Example

Traces client and server calls via interceptors.

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

### Run client

```sh
go run ./client
```
