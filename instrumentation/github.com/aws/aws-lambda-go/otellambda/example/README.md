# aws/aws-lambda-go instrumentation example

:warning: Deprecated: otellambda has no Code Owner.

A simple example to demonstrate the AWS Lambda for Go instrumentation. In this example, container `aws-lambda-client` initializes an S3 client and an HTTP client and runs 2 basic operations: `listS3Buckets` and `GET`.


These instructions assume you have
[docker-compose](https://docs.docker.com/compose/) installed and setup, and [AWS credential](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) configured.

1. From within the `example` directory, bring up the project by running:

    ```sh
    docker-compose up --detach
    ```

2. The instrumentation works with a `stdout` exporter. The example pulls this output from AWS and outputs back to stdout. 
   To inspect the output (following build output), you can run:

    ```sh
    docker-compose logs
    ```
3. After inspecting the client logs, the example can be cleaned up by running:

    ```sh
    docker-compose down
    ```

Note: Because the example runs on AWS Lambda, a handful of resources are created in AWS by the 
      example. The example will automatically destroy any resources it makes; however, if you
      terminate the container before it completes you may have leftover resources in AWS. Should
      you terminate the container early, run the below command to ensure all AWS resources are cleaned up: 

```sh
./manualAWSCleanup.sh
```
