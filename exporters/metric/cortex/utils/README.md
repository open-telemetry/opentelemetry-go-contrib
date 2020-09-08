# Configuration Utilities

This module allows users to create a `Config` struct from a YAML file. This functionality
is separate from the main module as it relies on [Viper](https://github.com/spf13/viper),
a large dependency that users may not wish to install.

## Usage
```go
// Create a custom HTTP client.
client := http.DefaultClient

// Search for a YAML file named config.yml in "." and "./configs" and use it to create a
// Config struct with the provided HTTP client.
config, err := NewConfig("config.yml", WithFilepath("./configs"), WithClient(client))
if err != nil {
    // Handle error
}

// Use newly created Config struct!
```



## Functionality

Users should use this module if they wish to use YAML files to configure the exporter. The
module provides the following functions:

```go
1. func NewConfig(filename string, opts ...Option) (*cortex.Config, error)
```
`NewConfig` is the primary function of this module. It create and returns a new
`cortex.Config` struct. By default, it searches for a YAML file (`.yaml` or `.yml`) in the
directory the function was called from. However, users can also provide multiple `Option`s
to add new filepaths to search for the YAML file in, a custom HTTP client, or an alternate
filesystem to use.

```go
1. func WithFilepath(filepath string) Option
```

`WithFilepath` adds a new filepath that `Viper` will search for YAML files in. This can be
called multiple times to add different filepaths. The local directory is searched by
default.


```go
1. func WithClient(client *http.Client) Option
```
`WithClient` sets the a custom HTTP client inside the Config struct. This is useful if the
provided configuration options are insufficient or if the user wants to customize other
HTTP client settings. For example, the client can be used to set up custom authentication.

```go
1. func WithFilesystem(fs afero.Fs) Option
```
`WithFilesystem` allows users to specify which filesystem `Viper` should search for the
YAML file in. This `Option` is used in the `config_utils_test.go` to search an in-memory
filesystem for created test files.
