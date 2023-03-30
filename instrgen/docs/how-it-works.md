## OpenTelemetry Go Source Automatic Instrumentation - How it works

`instrgen` adds OpenTelemetry instrumentation to source code by directly modifying it.
It uses the AST (Abstract Syntax Tree) representation of the code to determine its operational flow and injects necessary OpenTelemetry functionality into the AST.

The AST modification algorithm is the following:
1. Search for the entry point: a function definition with `AutotelEntryPoint()`.
2. Build the call graph. Traverse all calls from the entry point through all function definitions.
3. Inject OpenTelemetry instrumentation into functions bodies.
4. Context propagation. Adding an additional context parameter to all function declarations and function call expressions that are visible
   (it will not add a context argument to call expressions if they are not reachable from the entry point).
![image info](./flow.png)