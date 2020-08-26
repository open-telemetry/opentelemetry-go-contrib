# gomemcache instrumentation example

A simple example to demonstrate gomemcache client tracing instrumentation. It consists of two containers - `memcached-server`, which initializes and runs the Memcached server for this example, and `gomemcache-client`, which is the instrumented client.

In the example, the client invokes function `doMemcacheOperations()`, which is wrapped in a span. From within the function, the client will do a few example operations (add, get, delete with an intentional error) and cleans up the entries by calling `DeleteAll`.

These instructions expect you to have
[docker-compose](https://docs.docker.com/compose/) installed.

# Running the example

1. From within the `example` directory, bring up the project by running:

    ```sh
    docker-compose up --detach
    ```

2. The instrumentation works with a `stdout` exporter, meaning the spans should be visible in the output of the `gomemcache-container`. To inspect the output, you can run:

    ```sh
    docker-compose logs gomemcache-client
    ```

    In the log, total of 5 spans should appear - the parent span `test-operations` and 4 child spans, each corresponding to one client operation. Additionally, the `Delete` operation span should also include `StatusCode` and `StatusMessage`, as this operation intentionally leads to an error.

3. After inspecting the client logs, the example can be cleaned up by running:

    ```sh
    docker-compose down
    ```