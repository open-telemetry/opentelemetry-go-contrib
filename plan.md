# Fix otelgin negative metric sizes for unknown request bodies (#9054)

## Goal
Prevent `otelgin` from recording negative metric values (`-1`) when handling HTTP requests/responses with unknown body sizes (e.g. chunked encoding where `ContentLength == -1`).

## Implementation Steps
- [ ] 1. Update `RecordMetrics` in `internal/shared/semconv/server.go.tmpl` and generated `instrumentation/github.com/gin-gonic/gin/otelgin/internal/semconv/server.go` to guard `requestBodySizeHistogram` and `responseBodySizeHistogram` with `if md.RequestSize >= 0` and `if md.ResponseSize >= 0`.
- [ ] 2. Add unit test in `instrumentation/github.com/gin-gonic/gin/otelgin/gin_test.go` verifying that chunked / unknown length requests do not record negative metrics.
- [ ] 3. Run `gofmt` and `go test` for `otelgin` to ensure 100% passing tests.
- [ ] 4. Commit with DCO sign-off (`git commit -s`), push branch `fix/otelgin-negative-metric-sizes` to fork, and provide PR details.

## Implementation Progress
- [ ] Pending
