# OpenTelemetry Go SDK Prometheus Remote Write Exporter for Cortex

This module contains an exporter that sends cumulative metrics data from the OpenTelemetry
Go SDK to [Cortex](https://cortexmetrics.io/) using the Prometheus Remote Write API. While
it is aimed at Cortex, it should work with other backends that ingest data with the same
API.

This exporter is push-based and integrates with the OpenTelemetry Go SDK's [push
Controller](https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/metric/controller/push/push.go).
The Controller periodically collects data and passes it to this exporter. The exporter
then converts this data into
[`TimeSeries`](https://prometheus.io/docs/concepts/data_model/), a format that Cortex
accepts, and sends it to Cortex through HTTP POST requests. The request body is formatted
according to the protocol defined by the Prometheus Remote Write API. See Prometheus's
[remote storage integration
documentation](https://prometheus.io/docs/prometheus/latest/storage/#remote-storage-integrations)
for more details on the Remote Write API.

See the `example` submodule for a working example of this exporter.

Table of Contents
=================
   * [OpenTelemetry Go SDK Prometheus Remote Write Exporter for Cortex](#opentelemetry-go-sdk-prometheus-remote-write-exporter-for-cortex)
   * [Table of Contents](#table-of-contents)
      * [Installation](#installation)
      * [Setting up the Exporter](#setting-up-the-exporter)
      * [Configuring the Exporter](#configuring-the-exporter)
      * [Securing the Exporter](#securing-the-exporter)
         * [Authentication](#authentication)
         * [TLS](#tls)
      * [Instrument to Aggregation Mapping](#instrument-to-aggregation-mapping)
      * [Error Handling](#error-handling)
      * [Retry Logic](#retry-logic)
      * [Design Document](#design-document)
      * [Future Enhancements](#future-enhancements)

## Installation

```bash
go get -u go.opentelemetry.io/contrib/exporters/metric/cortex
```

## Setting up the Exporter

Users only need to call the `InstallNewPipeline` function to setup the exporter. It
requires a `Config` struct and returns a push Controller and error. If the error is nil,
the setup is successful and the user can begin creating instruments. No other action is
needed.

```go
// Create a Config struct named `config`.

controller, err := cortex.InstallNewPipeline(config)
if err != nil {
    return err
}
defer controller.Stop(context.Background())

// Make instruments and record data using `global.MeterProvider`.
```

## Configuring the Exporter

The Exporter requires certain information, such as the endpoint URL and push interval
duration, to function properly. This information is stored in a `Config` struct, which is
passed into the Exporter during the setup pipeline.

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

This is sourced from the Prometheus Remote Write Configuration
[documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write).

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

## Securing the Exporter

### Authentication
The exporter provides two forms of authentication which are shown below. Users can add
their own custom authentication by providing their own HTTP Client through the `Config`
struct and customizing it as needed.

1. Basic Authentication
   ```go
    // Basic Authentication properties in the Config struct.
    cortex.Config{
      // ...
      BasicAuth: map[string]string{
        "username":      "user",
        "password":      "password",
        "password_file": "passwordFile",
    }
   ```
   ```yaml
    # Basic Authentication properties in the YAML file.
    basic_auth:
      username: user
      password: password
      password_file: passwordfile
   ```
    Basic authentication sets a HTTP Authorization header containing a `base64` encoded
    username/password pair. See [RFC 7617](https://tools.ietf.org/html/rfc7617) for more
    information. Note that the password and password file are mutually exclusive. The
    `Config` struct's `Validate()` method will return an error if both are set.

2. Bearer Token Authentication
   ```go
    // Bearer Token Authentication properties in the Config struct.
   cortex.Config{
      BearerToken:     "token",
      BearerTokenFile: "tokenfile",
    }
   ```
   ```yaml
    # Bearer Token Authentication properties in the YAML file.
    bearer_token: token
    bearer_token_file: tokenfile
   ```
    Bearer token authentication sets a HTTP Authorization header containing a bearer token.
    See [RFC 6750](https://tools.ietf.org/html/rfc6750) for more information. Note that the
    bearer token and bearer token file are mutually exclusive. The `Config` struct's
    `Validate()` method will return an error if both are set.

### TLS
Users can add TLS to the exporter's HTTP Client through the `Config` struct by providing
certificate and key files. The certificate type does not matter. See `TestBuildClient()`
and `TestMutualTLS()` in `auth_test.go` for an example of how TLS can be used with the
exporter. `TestMutualTLS()` checks certificates between the exporter and a server both
ways.

```go
// TLS properties in the Config struct.
cortex.Config{
  TLSConfig: map[string]string{
    "ca_file":              "cafile",
    "cert_file":            "certfile",
    "key_file":             "keyfile",
    "server_name":          "server",
    "insecure_skip_verify": "0",
  },
}
```
```yaml
# TLS properties in the YAML file.
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
```

## Instrument to Aggregation Mapping
The exporter uses the `simple` selector's `NewWithHistogramDistribution()`. This means
that instruments are mapped to aggregations as shown in the table below.

| Instrument        | Aggregation |
|-------------------|-------------|
| Counter           | Sum         |
| UpDownCounter     | Sum         |
| ValueRecorder     | Histogram   |
| SumObserver       | Sum         |
| UpDownSumObserver | Sum         |
| ValueObserver     | Histogram   |

Although only the `Sum` and `Histogram` aggregations are currently being used, the
exporter supports 5 different aggregations:
1. `Sum`
2. `LastValue`
3. `MinMaxSumCount`
4. `Distribution`
5. `Histogram`

## Error Handling
In general, errors are returned to the calling function / method. Eventually, errors make
their way up to the push Controller where it calls the exporter's `Export()` method. The
push Controller passes the errors to the OpenTelemetry Go SDK's global error handler. 

The exception is when the exporter fails to send an HTTP request to Cortex. Regardless of
status code, the error is ignored. See the retry logic section below for more details.

## Retry Logic
The exporter does not implement any retry logic since the exporter sends cumulative
metrics data, which means that data will be preserved even if some exports fail. 

For example, consider a situation where a user increments a `Counter` instrument 5 times
and an export happens between each increment. If the exports happen like so:
```
  SUCCESS FAIL FAIL SUCCESS SUCCESS
  1       2    3    4       5
```
Then the received data will be:
```
1 4 5
```

The end result is the same since the aggregations are cumulative.

## Design Document

[Design Document](https://github.com/open-o11y/docs/blob/main/go-prometheus-remote-write/design-doc.md)

The document is not in this module as it contains large images which will increase the
size of the overall repo significantly.

## Future Enhancements
* Add configuration option for different selectors 

   Users may not want to use the default Histogram selector and should be able to choose
  which selector they want to use.
