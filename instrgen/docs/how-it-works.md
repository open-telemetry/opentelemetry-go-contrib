## OpenTelemetry Go Source Automatic Instrumentation - How it works

`instrgen` adds OpenTelemetry instrumentation to source code by directly modifying it.
It uses the AST (Abstract Syntax Tree) representation of the code to determine its operational flow and injects necessary OpenTelemetry functionality into the AST.

`instrgen` utilizes toolexec golang compiler switch. It means that it has access to all files
that takes part in the compilation process.

The AST modification algorithm is the following:
1. Rewrites go runtime package in order to provide correct context propagation.
2. Inject OpenTelemetry instrumentation into functions bodies.
