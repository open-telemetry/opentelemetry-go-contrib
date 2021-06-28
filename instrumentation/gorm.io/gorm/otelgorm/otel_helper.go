package otelgorm

import (
	"context"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	_systemTagKey       = "db.system"
	_tableTagKey        = "db.sql.table"
	_errorTagKey        = "db.error"
	_resultLogKey       = "db.result"
	_sqlLogKey          = "db.statement"
	_rowsAffectedLogKey = "db.rows.affected"
)

var (
	opentelemetrySpanKey = "opentelemetry:span"
	json                 = jsoniter.ConfigCompatibleWithStandardLibrary
)

func (p otelPlugin) injectBefore(db *gorm.DB, op operationName) {
	// make sure context could be used
	if db == nil {
		return
	}

	if db.Statement == nil || db.Statement.Context == nil {
		db.Logger.Error(context.TODO(), "could not inject sp from nil Statement.Context or nil Statement")
		return
	}
	_, sp := p.opt.tracer.Start(db.Statement.Context, "GORM "+op.String())
	db.InstanceSet(opentelemetrySpanKey, sp)
}

func (p otelPlugin) extractAfter(db *gorm.DB) {
	// make sure context could be used
	if db == nil {
		return
	}
	if db.Statement == nil || db.Statement.Context == nil {
		db.Logger.Error(context.TODO(), "could not extract sp from nil Statement.Context or nil Statement")
		return
	}

	// extract sp from db context
	//sp := opentelemetry.SpanFromContext(db.Statement.Context)
	v, ok := db.InstanceGet(opentelemetrySpanKey)
	if !ok || v == nil {
		return
	}

	sp, ok := v.(trace.Span)
	if !ok || sp == nil {
		return
	}
	defer sp.End()

	// tag and log fields we want.
	tag(sp, db)
	evnet(sp, db, p.opt.logResult, p.opt.logSqlParameters)
}

// tag called after operation
func tag(sp trace.Span, db *gorm.DB) {
	if err := db.Error; err != nil {
		sp.SetAttributes(attribute.Bool(_errorTagKey, true))
	}

	sp.SetAttributes(
		attribute.String(_systemTagKey, db.Name()),
		attribute.String(_tableTagKey, db.Statement.Table),
	)
}

// evnet called after operation
func evnet(sp trace.Span, db *gorm.DB, verbose bool, logSqlVariables bool) {
	attrs := make([]attribute.KeyValue, 0, 4)
	attrs = appendSql(attrs, db, logSqlVariables)
	attrs = append(attrs, attribute.Int64(_rowsAffectedLogKey, db.Statement.RowsAffected))

	// log error
	if err := db.Error; err != nil {
		sp.RecordError(db.Error)
	}

	if verbose && db.Statement.Dest != nil {
		// DONE(@yeqown) fill result fields into span log
		// FIXED(@yeqown) db.Statement.Dest still be metatable now ?
		v, err := json.Marshal(db.Statement.Dest)
		if err == nil {
			attrs = append(attrs, attribute.String(_resultLogKey, *(*string)(unsafe.Pointer(&v))))
		} else {
			db.Logger.Error(context.Background(), "could not marshal db.Statement.Dest: %v", err)
		}
	}

	sp.AddEvent("sql", trace.WithAttributes(attrs...))
}

func appendSql(attrs []attribute.KeyValue, db *gorm.DB, logSqlVariables bool) []attribute.KeyValue {
	if logSqlVariables {
		attrs = append(attrs, attribute.String(_sqlLogKey,
			db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)))
	} else {
		attrs = append(attrs, attribute.String(_sqlLogKey, db.Statement.SQL.String()))
	}
	return attrs
}
