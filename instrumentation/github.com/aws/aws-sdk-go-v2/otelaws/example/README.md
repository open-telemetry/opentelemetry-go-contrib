# aws/aws-sdk-go-v2 instrumentation example

A simple example to demonstrate aws sdk v2 for Go instrumentation. In this example, container `aws-sdk-client` initializes a s3 client and a dynamodb client and runs 2 basic operations: listS3Buckets and listDynamodbTables. 


These instructions assume you have
[docker-compose](https://docs.docker.com/compose/) installed and setup [aws credential](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) in local.

1. From within the `example` directory, bring up the project by running:

    ```sh
    docker-compose up --detach
    ```

2. The instrumentation works with a `stdout` exporter. To inspect the output, you can run:

    ```sh
    docker-compose logs
    ```
3. After inspecting the client logs, the example can be cleaned up by running:

    ```sh
    docker-compose down
    ```
