# SkyWalking Propagator

This package provides a SkyWalking propagator for OpenTelemetry Go.

## SW8 Header Format

The implementation follows the official SkyWalking v3 Cross Process Propagation Headers Protocol:

```
sw8: {sample}-{trace-id}-{parent-trace-segment-id}-{parent-span-id}-{parent-service}-{parent-service-instance}-{parent-endpoint}-{target-address}
```

Where:

- **Field 0**: Sample flag ("1" if sampled, "0" if context exists but may be ignored)
- **Field 1**: Trace ID (Base64 encoded hex string, globally unique)
- **Field 2**: Parent trace segment ID (Base64 encoded hex string, globally unique)
- **Field 3**: Parent span ID (integer, begins with 0, points to parent span in parent trace segment)
- **Field 4**: Parent service (Base64 encoded, max 50 UTF-8 characters)
- **Field 5**: Parent service instance (Base64 encoded, max 50 UTF-8 characters)
- **Field 6**: Parent endpoint (Base64 encoded, max 150 UTF-8 characters, operation name of first entry span)
- **Field 7**: Target address (Base64 encoded, network address used on client end)

## SW8-Correlation Header Format

The propagator supports SkyWalking correlation headers following the official v1 specification:

```
sw8-correlation: base64(key1):base64(value1),base64(key2):base64(value2)
```

Key features:

- **Base64 Encoding**: Both keys and values are Base64 encoded as per official specification
- **Limits**: Maximum 3 keys, each value maximum 128 bytes (before encoding)
- **Integration**: Automatic extraction from and injection into OpenTelemetry baggage
- **Error Handling**: Graceful handling of malformed headers and encoding errors

## SW8-X Extension Header Format

The propagator also supports SkyWalking extension headers following the v3 specification:

```
sw8-x: {tracing-mode}-{timestamp}
```

Current implementation:

- **Field 1**: Tracing Mode ("0" = normal analysis, "1" = skip analysis)
- **Field 2**: Timestamp for async RPC latency calculation (milliseconds since epoch)
- **Default**: Uses "0" (normal tracing mode) with placeholder timestamp (" ")

## Usage

### Basic Usage

```go
import "go.opentelemetry.io/contrib/propagators/skywalking"

// Create propagator
propagator := skywalking.Skywalking{}

// Use with OpenTelemetry
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
    propagator,
    propagation.TraceContext{},
    propagation.Baggage{},
))
```

### Correlation Data Usage

The propagator automatically handles SkyWalking correlation data through OpenTelemetry baggage using BASE64 encoding:

```go
import (
    "go.opentelemetry.io/otel/baggage"
    "go.opentelemetry.io/contrib/propagators/skywalking"
)

// Add correlation data to baggage
member1, _ := baggage.NewMember("service.name", "web-service")
member2, _ := baggage.NewMember("user.id", "user123")
bags, _ := baggage.New(member1, member2)

// Create context with baggage
ctx := baggage.ContextWithBaggage(context.Background(), bags)

// The propagator will automatically inject correlation data into sw8-correlation header
// Format: base64("service.name"):base64("web-service"),base64("user.id"):base64("user123")
propagator := skywalking.Skywalking{}
carrier := make(propagation.MapCarrier)
propagator.Inject(ctx, carrier)

// On the receiving side, correlation data is automatically extracted into baggage
extractedCtx := propagator.Extract(context.Background(), carrier)
extractedBags := baggage.FromContext(extractedCtx)
serviceName := extractedBags.Member("service.name").Value() // "web-service"
```

The propagator enforces the official specification limits:

- Maximum 3 correlation keys per request
- Maximum 128 bytes per value (before BASE64 encoding)
- Automatic BASE64 encoding/decoding for safe transport

The propagator uses default "unknown" values for service metadata fields in the SW8 header, following the stateless design principle.

### Tracing Mode Control

The propagator supports SkyWalking tracing mode control through context utilities:

```go
import "go.opentelemetry.io/contrib/propagators/skywalking"

// Set skip analysis mode
ctx = skywalking.WithTracingMode(ctx, skywalking.TracingModeSkipAnalysis)

// Inject includes SW8-X header with tracing mode
propagator.Inject(ctx, carrier)

// Extract preserves tracing mode in context
extractedCtx := propagator.Extract(context.Background(), carrier)
mode := skywalking.TracingModeFromContext(extractedCtx)
```

### Timestamp Support for Transmission Latency

The propagator supports timestamps in SW8-X headers for transmission latency calculation:

```go
import (
    "time"
    "go.opentelemetry.io/contrib/propagators/skywalking"
)

// Set timestamp before sending request (milliseconds since epoch)
timestamp := time.Now().UnixMilli()
ctx = skywalking.WithTimestamp(ctx, timestamp)

// Inject includes SW8-X header with timestamp
propagator.Inject(ctx, carrier)

// Extract preserves timestamp in context for latency calculation
extractedCtx := propagator.Extract(context.Background(), carrier)
receivedTimestamp := skywalking.TimestampFromContext(extractedCtx)
if receivedTimestamp > 0 {
    latency := time.Now().UnixMilli() - receivedTimestamp
    // Use latency for monitoring/observability
}
```

## Specification Reference

This implementation is based on the official [SkyWalking Cross Process Propagation Headers Protocol v3](https://skywalking.apache.org/docs/main/latest/en/api/x-process-propagation-headers-v3/)
