# Dynamic config service example

Right now the generated protobuf .go file we're using is in our fork of the repo. Eventually we want to import in from opentelemetry-proto.

Additionally, right now (June 18, 2020), the collector received an update that makes it incompatible with metric exporting from the SDK. We will need to export to an older version of the collector.

### Expected behaviour for this example
Initial period of 10 seconds. After about 10 seconds, new config will be read, and collection period is changed to 1 second.

You can change expected behaviour by editing the config served by the dummy configuration service in `./server/server.go`.

### Setup and run collector

```sh
# Clone repo
git clone https://github.com/open-telemetry/opentelemetry-collector.git

cd opentelemetry-collector

# Checkout proper commit
git branch compatible-master 746db761d19ed12ac2278cdfe7f30826a5ba6257 && git checkout compatible-master

# Build collector binary
make otelcol

# Run collector using example-otlp-config.yaml
# Assume opentelemetry-go is in the home directory
./bin/otelcol_linux_amd64 --config ~/opentelemetry-go-contrib/exporters/metric/dynamicconfig/example/example-otlp-config.yaml
```

### Run server

```sh
go run ./server
```

### Run sdk

```sh
go run ./
```
