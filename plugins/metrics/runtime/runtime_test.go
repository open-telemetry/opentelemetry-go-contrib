package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/api/global"
)

func TestRuntime(t *testing.T) {
	meter := global.Meter("test")
	r := NewRuntime(meter, time.Second)
	err := r.Start()
	assert.NoError(t, err)
	time.Sleep(time.Second)
	r.Stop()
}
