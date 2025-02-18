# Config example

This is an end-to-end example showing how to use the new `config` package, allowing the OTel SDK to be configured using an external configuration file.

## Usage

1. start a backend that is capable of receiving logs, metrics, and traces in OTLP format. If you are unsure, [`otel-tui`](https://github.com/ymtdzzz/otel-tui) is a simple option that works.
2. configure the provided `otel.yaml` to point to your backend. If you are using a local backend, the provided configuration file should work out-of-the-box.
3. Run `go run ./`
