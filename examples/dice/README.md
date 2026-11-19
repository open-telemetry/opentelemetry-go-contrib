# Dice example

This is the foundation example for [Getting Started](https://opentelemetry.io/docs/languages/go/getting-started/) with OpenTelemetry.

Below, you will see instructions on how to run this application, either with or without instrumentation.

## Usage

The `run.sh` script accepts one argument to determine which example to run:

- `uninstrumented`
- `instrumented`

Both the instrumented and uninstrumented versions accept an environment variable, `APPLICATION_PORT` (defaults to `8080`).

### Endpoints

Once the application is running, you can roll the dice by sending a request to `/rolldice`:

```http
GET /rolldice
```

#### Query Parameters

- `player` *(optional)*: The name of the player rolling the dice (e.g., `/rolldice?player=Arun`).
- `rolls` *(optional)*: The number of times to roll the dice (e.g., `/rolldice?rolls=3`).

**Note:** The `/rolldice/{player}` path parameter has been removed. Use the `player` query parameter on `/rolldice` instead.

### Running the Uninstrumented Example

The uninstrumented example is a very simple dice application, without OpenTelemetry instrumentation.

To run the uninstrumented example, execute:

```bash
./run.sh uninstrumented
```

### Running the Instrumented Example

The instrumented example is exactly the same application, which includes OpenTelemetry instrumentation.

To run the instrumented example, execute:

```bash
./run.sh instrumented
```
