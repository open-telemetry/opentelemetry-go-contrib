package runtime

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/api/global"
)

func TestRuntime(t *testing.T) {
	meter := global.Meter("test")
	cancelRuntime := Runtime(context.Background(), meter, time.Second)
	cancelRuntime()
}
