package zap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testBodyString = "log message"
	testSeverity   = log.SeverityInfo
)

type spyLogger struct {
	embedded.Logger
	Context context.Context
	Record  log.Record
}

func (l *spyLogger) Emit(ctx context.Context, r log.Record) {
	l.Context = ctx
	l.Record = r
}

func NewTestOtelLogger(log log.Logger) zapcore.Core {
	return &OtelZapCore{
		logger: log,
	}
}
func TestZapCore(t *testing.T) {
	spy := &spyLogger{}
	logger := zap.New(NewTestOtelLogger(spy))
	logger.Info(testBodyString, zap.String("username", "johndoe"))

	assert.Equal(t, testBodyString, spy.Record.Body().AsString())
	assert.Equal(t, testSeverity, spy.Record.Severity())
	assert.Equal(t, 1, spy.Record.AttributesLen())
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "username", string(kv.Key))
		assert.Equal(t, "johndoe", kv.Value.AsString())
		return true
	})

	childlogger := logger.With(zap.String("workplace", "otel"))
	childlogger.Info(testBodyString)
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "workplace", string(kv.Key))
		assert.Equal(t, "otel", kv.Value.AsString())
		return true
	})

}
