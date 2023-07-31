# OpenTelemetry Go Source Automatic Instrumentation

This package provides a code generation utility that instruments existing source code with [OpenTelemetry].
If you are looking for more details about internal working, see [How it works](./docs/how-it-works.md).

## Project Status

:construction: This package is currently work in progress.

## How to use it

In order to instrument your project you have to add following call in your entry point function, usually main
(you can look at testdata directory for reference) and invoke instrgen tool.

```
func main() {
    rtlib.AutotelEntryPoint()
```

Instrgen requires three parameters: command, path to project and package(s) pattern we
would like to instrument.

```
./instrgen --inject [path to your go project] [package(s) pattern]
```

Below concrete example with one of test instrumentation that is part of the project.

```
./instrgen --inject ./testdata/basic ./...
```

```./...``` works like wildcard in this case and it will instrument all packages in this path, but it can be invoked with
specific package as well.

### Compatibility

The `instrgen` utility is based on the Go standard library and is platform agnostic.

[OpenTelemetry]: https://opentelemetry.io/
