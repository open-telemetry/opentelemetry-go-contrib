## OpenTelemetry Go Source Automatic Instrumentation - How it works

`instrgen` driver modifies AST (Abstract Syntax Tree) in order to inject necessary opentelemetry calls.

There are few passes during execution.
1. Searching for entry point, a function definition with ```AutotelEntryPoint__()``` call.
2. Building call graph. Traversing all calls from entry point through all function definitions.
3. Injecting open telemetry calls into functions bodies.
   is before context propagation due to fact of changing type signatures by context propagation)
4. Context propagation. Adding additional context parameter to all function declarations and function call expressions that are visible
   (it will not add context argument to call expression without having visible function declaration).
![image info](./flow.png)