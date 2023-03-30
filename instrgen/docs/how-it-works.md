## OpenTelemetry Go Source Automatic Instrumentation - How it works

`instrgen` adds OpenTelemetry instrumentation to source code by directly modifying it.
It uses the AST (Abstract Syntax Tree) representation of the code to determine its operational flow and injects necessary OpenTelemetry functionality into the AST.

The AST modification algorithm is the following:
1. Search for the entry point: a function definition with `AutotelEntryPoint()`.
2. Building call graph. Traversing all calls from entry point through all function definitions.
3. Injecting open telemetry calls into functions bodies.
4. Context propagation. Adding additional context parameter to all function declarations and function call expressions that are visible
   (it will not add context argument to call expression without having visible function declaration).
![image info](./flow.png)