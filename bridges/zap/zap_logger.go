package zap

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.uber.org/zap/zapcore"
)

const (
	bridgeName    = "go.opentelemetry.io/contrib/bridge/zapcore"
	bridgeVersion = "0.0.1-alpha"
)

type OtelZapCore struct {
	logger log.Logger
	attr   []log.KeyValue
}

var (
	_ zapcore.Core = (*OtelZapCore)(nil)
)

// this function creates a new zapcore.Core that can be used with zap.New()
// this instance will translate zap logs to opentelemetry logs and export them

func NewOtelZapCore(lp log.LoggerProvider, opts ...log.LoggerOption) zapcore.Core {
	if lp == nil {
		// Do not panic.
		lp = noop.NewLoggerProvider()
	}
	// these options
	return &OtelZapCore{
		logger: lp.Logger(bridgeName,
			log.WithInstrumentationVersion(bridgeVersion),
		),
	}
}

func (o *OtelZapCore) Enabled(zapcore.Level) bool {
	return true
	//return o.logger.Enabled()
}

// return new zapcore with provided attr
func (o *OtelZapCore) With(fields []zapcore.Field) zapcore.Core {
	clone := o.clone()
	enc := NewOtelObjectEncoder()
	for i := range fields {
		fields[i].AddTo(enc)
	}
	clone.attr = enc.cur
	return clone
}

// TODO
func (o *OtelZapCore) Sync() error {
	return nil
}

func (o *OtelZapCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// should change zap level to otel log level?
	if o.Enabled(ent.Level) {
		return ce.AddCore(ent, o)
	}
	return ce
}

func (o *OtelZapCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var attr []log.KeyValue

	// add fields to attr
	enc := NewOtelObjectEncoder()
	for i := range fields {
		fields[i].AddTo(enc)
	}
	attr = enc.cur

	// we create record here to avoid heap allocation
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetBody(log.StringValue(ent.Message))

	// should confirm this
	sevOffset := 4*(ent.Level+2) + 1
	r.SetSeverity(log.Severity(ent.Level + sevOffset))

	// append attributes received from from parent logger``
	addattr := append(attr, o.attr...)

	if len(addattr) > 0 {
		r.AddAttributes(addattr...)
	}
	// need to check how to pass context here
	ctx := context.Background()
	o.logger.Emit(ctx, r)
	return nil
}

func (o *OtelZapCore) clone() *OtelZapCore {
	return &OtelZapCore{
		logger: o.logger,
	}
}

// func convertAttr(f zapcore.Field) log.KeyValue {
// 	val := convertValue(f)
// 	return log.KeyValue{Key: f.Key, Value: val}
// }

// // how to check if all types are covered?
// // this does not cover all Field types yet
// func convertValue(f zapcore.Field) log.Value {
// 	switch f.Type {
// 	case zapcore.ArrayMarshalerType:
// 		s := &sliceArrayEncoder{}
// 		f.Interface.(zapcore.ArrayMarshaler).MarshalLogArray(s)
// 		// Debug statement
// 		fmt.Println(s.elems)
// 		return log.StringValue("random")
// 		//return log.SliceValue(f.Interface.(zapcore.ArrayMarshaler))
// 	case zapcore.ObjectMarshalerType:
// 		fmt.Println("hello")
// 		s := NewOtelObjectEncoder()
// 		err := f.Interface.(zapcore.ObjectMarshaler).MarshalLogObject(s)
// 		fmt.Println(log.MapValue(s.cur...).AsMap(), err)
// 		return log.StringValue("ObjectMarshalerType")
// 		// f.Interface.(ObjectMarshaler))
// 	case zapcore.InlineMarshalerType:
// 		return log.StringValue("InlineMarshalerType")
// 		//return log f.Interface.(ObjectMarshaler).MarshalLogObject(enc)
// 	case zapcore.BinaryType:
// 		return log.BytesValue(f.Interface.([]byte))
// 	case zapcore.BoolType:
// 		return log.BoolValue(f.Integer == 1)
// 	case zapcore.ByteStringType:
// 		return log.BytesValue(f.Interface.([]byte))
// 	case zapcore.Complex128Type:
// 		// return log.StringValue("complex128type")
// 		return log.Float64Value(f.Interface.(float64))
// 		//return log.Float64Value(f.Interface.(complex128))
// 	case zapcore.Complex64Type:
// 		// return log.StringValue("Complex64Type")
// 		return log.Float64Value(f.Interface.(float64))
// 	case zapcore.DurationType:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Float64Type:
// 		return log.Float64Value(math.Float64frombits(uint64(f.Integer)))
// 	case zapcore.Float32Type:
// 		return log.Float64Value(math.Float64frombits(uint64(f.Integer)))
// 	case zapcore.Int64Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Int32Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Int16Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Int8Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.StringType:
// 		return log.StringValue(f.String)
// 	case zapcore.TimeType:
// 		// if f.Interface != nil {
// 		// 	return time.Unix(0, f.Integer).In(f.Interface.(*time.Location))
// 		// } else {
// 		// 	// Fall back to UTC if location is nil.
// 		// 	return time.Unix(0, f.Integer)
// 		// }
// 		return log.Int64Value(f.Integer)
// 	case zapcore.TimeFullType:
// 		return log.StringValue("TimeFullType")
// 	// 	enc.AddTime(f.Key, f.Interface.(time.Time))
// 	case zapcore.Uint64Type:
// 		asInt64 := f.Integer
// 		if asInt64 > 0 {
// 			return log.Int64Value((asInt64))
// 		} else {
// 			return log.StringValue("overflowed uint64")
// 		}
// 	case zapcore.Uint32Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Uint16Type:
// 		return log.Int64Value(f.Integer)
// 	case zapcore.Uint8Type:
// 		return log.Int64Value(f.Integer)

// 		// how to handle these types
// 		// case zapcore.SkipType:
// 		// 	break
// 	case zapcore.UintptrType:
// 		return log.Int64Value(int64(f.Integer))
// 	// case ReflectType:
// 	// 	err = enc.AddReflected(f.Key, f.Interface)
// 	// case NamespaceType:
// 	// 	enc.OpenNamespace(f.Key)
// 	// case StringerType:
// 	// 	err = encodeStringer(f.Key, f.Interface, enc)
// 	// case ErrorType:
// 	// 	err = encodeError(f.Key, f.Interface.(error), enc)
// 	default:
// 		panic(fmt.Sprintf("unhandled attribute kind: %c", f.Type))
// 	}
// }
