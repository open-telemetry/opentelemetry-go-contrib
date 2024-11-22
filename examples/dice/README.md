# Dice example

This is the foundation example for [Getting Started](https://opentelemetry.io/docs/languages/go/getting-started/) with OpenTelemetry.

Below, you will see instructions on how to run this application, either with or without instrumentation.

## Usage

The `run.sh` script accepts one argument to determine which example to run:

- `uninstrumented`
- `instrumented`

### Running the Uninstrumented Example

The uninstrumented example is a very simple dice application, without OpenTelemetry instrumentation.

To run the uninstrumented example, execute:

```bash
./run.sh uninstrumented
```

### Running the Instrumented Example

The instrumented example is exactly the same application, which includes OpenTelemetry instrumentation. 

To run the instrumented example, execute:

```bash
./run.sh instrumented
```