# OpenTelemetry Go Source Automatic Instrumentation

This package provides a code generation utility that instruments existing source code with [OpenTelemetry].
If you are looking for more details about internal working, see [How it works](./docs/how-it-works.md).

## Project Status

:construction: This package is currently work in progress.

## Build

From driver directory execute:

```
go build
```

## Prerequisites

`instrgen` driver utility needs to be on your PATH environment variable.

## How to use it

Instrgen has to be invoked from main module directory and
requires three parameters: command, directory (files from specified directory will be rewritten).

```
./driver --inject [file pattern] [replace input source] [entry point]
```

Below concrete example with one of test instrumentation that is part of the project.

```
driver --inject  /testdata/basic yes main.main
```

Above command will invoke golang compiler under the hood:

```
go build -work -a -toolexec driver
```

which means that the above command can be executed directly, however first `instrgen_cmd.json`
configuration file needs to be provided. This file is created internally by `driver` based on provided
command line.

Below example content of `instrgen_cmd.json`:

```
{
"ProjectPath": ".",
"FilePattern": "/testdata/basic",
"Cmd": "inject",
"Replace": "yes",
"EntryPoint": {
    "Pkg": "main",
    "FunName": "main"
 }
}
```

### Work in progress:

Library instrumentation:
- HTTP

### Compatibility

The `instrgen` utility is based on the Go standard library and is platform agnostic.

[OpenTelemetry]: https://opentelemetry.io/
