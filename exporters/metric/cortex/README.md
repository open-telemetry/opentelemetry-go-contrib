# OpenTelemetry Go SDK Cortex Exporter

Work in progress exporter to send data from the OpenTelemetry Go SDK to Cortex using the Prometheus
Remote Write API.

## Configuration

The Exporter needs certain information, such as the endpoint URL and push interval
duration, to function properly. This information is stored in a `Config` struct, which is
passed into the Exporter during the setup pipeline. 

### Creating the Config struct

Users can either create the struct manually or use a `utils` submodule in the package to
read settings from a YAML file into a new Config struct using `Viper`. Here are the
supported YAML properties as well as the Config struct that they map to.

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
```

```go
type Config struct {
	Endpoint        string            `mapstructure:"url"`
	RemoteTimeout   time.Duration     `mapstructure:"remote_timeout"`
	Name            string            `mapstructure:"name"`
	BasicAuth       map[string]string `mapstructure:"basic_auth"`
	BearerToken     string            `mapstructure:"bearer_token"`
	BearerTokenFile string            `mapstructure:"bearer_token_file"`
	TLSConfig       map[string]string `mapstructure:"tls_config"`
	ProxyURL        string            `mapstructure:"proxy_url"`
	PushInterval    time.Duration     `mapstructure:"push_interval"`
	Headers         map[string]string `mapstructure:"headers"`
	Client          *http.Client
}
```

The struct is used during the setup pipeline:

```go
// Create Config struct using utils module.
config, err := utils.NewConfig("config.yml")
if err != nil {
    return err
}

// Setup the exporter.
pusher, err := cortex.InstallNewPipeline(config)
if err != nil {
    return err
}

// Add instruments and start collecting data.
```
