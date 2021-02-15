# gorm instrumentation example

A simple example to demonstrate gorm tracing instrumentation. 

In the example, the client invokes function `doGormOperations()`, which is wrapped in a span. From within the function, the client will do a few example operations (create, get, delete, update) including the last one that generates an error.


# Running the example

1. From within the `example` directory, bring up the project by running:

    ```sh
    go run ./main.go
    ```

2. The instrumentation works with a `stdout` exporter, meaning the spans should be visible in stdoud.

   You should see several spans in the output, each corresponding to one client operation. Additionally, the last `SELECT` operation span should also include `StatusCode` and `StatusMessage`, as this operation intentionally leads to an error.

