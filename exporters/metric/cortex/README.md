# OpenTelemetry Go SDK Prometheus Remote Write Exporter for Cortex

This module contains an exporter that sends metrics data from the OpenTelemetry Go SDK to
[Cortex](https://cortexmetrics.io/) using the Prometheus Remote Write API. While it is
aimed at Cortex, it should work with other backends that ingest data with the same API.

This exporter is push-based and integrates with the OpenTelemetry Go SDK's [push
Controller](https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/metric/controller/push/push.go).
The Controller periodically collects data and passes it to this exporter. The exporter
then converts this data into
[`TimeSeries`](https://prometheus.io/docs/concepts/data_model/), a format that Cortex
accepts, and sends it to Cortex through HTTP POST requests. The request body is formatted
according to the protocol defined by the Prometheus Remote Write API.

See the `example` submodule for a working example of this exporter.

## Setting up the Exporter

Users only need to call the `InstallNewPipeline` function to setup the exporter. It
requires a `Config` struct and returns a push Controller and error. If the error is nil,
the setup is successful and the user can begin creating instruments. No other action is
needed.

```go
// Create a Config struct named `config`.

pusher, err := cortex.InstallNewPipeline(config)
if err != nil {
    return err
}

// Make instruments and record data.
```

## Configuring the Exporter

The Exporter requires certain information, such as the endpoint URL and push interval
duration, to function properly. This information is stored in a `Config` struct, which is
passed into the Exporter during the setup pipeline.

### Creating a Config struct

There are two options for creating a `Config` struct:

1. Use the `utils` submodule to read settings from a YAML file into a new `Config` struct
   * Call `utils.NewConfig(...)` to create the struct. More details can be found in the `utils` module's README
   * `Config` structs have a `Validate()` method that sets defaults and checks for errors,
     but it isn't necessary to call it as `utils.NewConfig()` does that already.

2. Create the `Config` struct manually
   * Users should call the `Config` struct's `Validate()` method to set default values and
     check for errors.

```go
// (Option 1) Create Config struct using utils module.
config, err := utils.NewConfig("config.yml")
if err != nil {
    return err
}

// (Option 2) Create Config struct manually.
configTwo := cortex.Config {
  Endpoint:      "http://localhost:9009/api/prom/push",
	RemoteTimeout: 30 * time.Second,
	PushInterval: 5 * time.Second,
	Headers: map[string]string{
		"test": "header",
	},
}
// Validate() should be called when creating the Config struct manually.
err = config.Validate()
if err != nil {
	return err
}
```

The Config struct supports many different configuration options. Here is the `Config`
struct definition as well as the supported YAML properties. The `mapstructure` tags are
used by the `utils` submodule.

```go
type Config struct {
	Endpoint            string            `mapstructure:"url"`
	RemoteTimeout       time.Duration     `mapstructure:"remote_timeout"`
	Name                string            `mapstructure:"name"`
	BasicAuth           map[string]string `mapstructure:"basic_auth"`
	BearerToken         string            `mapstructure:"bearer_token"`
	BearerTokenFile     string            `mapstructure:"bearer_token_file"`
	TLSConfig           map[string]string `mapstructure:"tls_config"`
	ProxyURL            string            `mapstructure:"proxy_url"`
	PushInterval        time.Duration     `mapstructure:"push_interval"`
	Quantiles           []float64         `mapstructure:"quantiles"`
	HistogramBoundaries []float64         `mapstructure:"histogram_boundaries"`
	Headers             map[string]string `mapstructure:"headers"`
	Client              *http.Client
}
```

<details>
<summary>Supported YAML Properties</summary>

```yaml
# The URL of the endpoint to send samples to.
url: <string>

# Timeout for requests to the remote write endpoint.
[ remote_timeout: <duration> | default = 30s ]

# Name of the remote write config, which if specified must be unique among remote write configs. The name will be used in metrics and logging in place of a generated value to help users distinguish between remote write configs.
[ name: <string>]

# Sets the `Authorization` header on every remote write request with the
# configured username and password.
# password and password_file are mutually exclusive.
basic_auth:
  [ username: <string>]
  [ password: <string>]
  [ password_file: <string> ]

# Sets the `Authorization` header on every remote write request with
# the configured bearer token. It is mutually exclusive with `bearer_token_file`.
[ bearer_token: <string> ]

# Sets the `Authorization` header on every remote write request with the bearer token
# read from the configured file. It is mutually exclusive with `bearer_token`.
[ bearer_token_file: /path/to/bearer/token/file ]

# Configures the remote write request's TLS settings.
tls_config:
  # CA certificate to validate API server certificate with.
  [ ca_file: <filename>]

  # Certificate and key files for client cert authentication to the   server.
  [ cert_file: <filename> ]
  [ key_file: <filename> ]

  # ServerName extension to indicate the name of the server.
  # https://tools.ietf.org/html/rfc4366#section-3.1
  [ server_name: <string> ]

  # Disable validation of the server certificate.
  [ insecure_skip_verify: <boolean> ]

# Optional proxy URL.
[ proxy_url: <string>]

# Quantiles for Distribution aggregations
[ quantiles: ]
  - <string>
  - <string>
  - ...

# Histogram Buckets
[ histogram_buckets: ]
  - <string>
  - <string>
  - ...
```
</details>

