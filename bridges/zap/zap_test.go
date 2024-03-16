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
	logger.Info(testBodyString, zap.Strings("key", []string{"1", "2"}))

	assert.Equal(t, testBodyString, spy.Record.Body().AsString())
	assert.Equal(t, testSeverity, spy.Record.Severity())
	assert.Equal(t, 1, spy.Record.AttributesLen())
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "key", string(kv.Key))
		assert.Equal(t, "1", kv.Value.AsSlice()[0].AsString())
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

type addr struct {
	IP   string
	Port int
}

type request struct {
	URL    string
	Listen addr
	Remote addr
}

func (a addr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("ip", a.IP)
	enc.AddInt("port", a.Port)
	return nil
}

func (r *request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("url", r.URL)
	zap.Inline(r.Listen).AddTo(enc)
	return enc.AddObject("remote", r.Remote)
}

// func TestObjectEncoder(t *testing.T) {
// 	spy := &spyLogger{}
// 	logger := zap.New(NewTestOtelLogger(spy))
// 	// logger.Info(testBodyString, zap.Strings("key", []string{"1", "2"}))
// 	req := &request{
// 		URL:    "/test",
// 		Listen: addr{"127.0.0.1", 8080},
// 		Remote: addr{"127.0.0.1", 31200},
// 	}
// 	// Use the ObjectValues field constructor when you have a list of
// 	// objects that do not implement zapcore.ObjectMarshaler directly,
// 	// but on their pointer receivers.
// 	logger.Info("new request, in nested object", zap.Object("req", req))
// 	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
// 		assert.Equal(t, "req", string(kv.Key))
// 		assert.Equal(t, req, kv.Value.AsMap())
// 		fmt.Println(kv.Value.AsString())
// 		return true
// 	})
// }
