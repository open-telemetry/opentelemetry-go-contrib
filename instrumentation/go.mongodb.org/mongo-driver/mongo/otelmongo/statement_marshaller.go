package otelmongo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TODO: Add a marshaller to whitelist attributes
// TODO: Add a marshaller to blacklist attributes

// StatementMarshaller is an interface that implements the method to marshal the raw bson command into the
// db.statement attribute. It can add othe attributes if needed
type StatementMarshaller interface {
	Marshal(command bson.Raw) []attribute.KeyValue
}

// NewLimitedStatementMarshaller creates a new limitedStatementMarshaller, that limits the size of the
// db.statement to the specified size.
// WARNING: This marshaller doesn't perform any sanitization of the statement and could leak sensitive
// information
func NewLimitedStatementMarshaller(limit int) StatementMarshaller {
	return &limitedStatementMarshaller{
		limit: limit,
	}
}

// limitedStatementMarshaller implements StatementMarshaller.
// WARNING: This implementation marshals the whole db statement without any sanitization
// and could lead to leaks of sensitive information.
type limitedStatementMarshaller struct {
	limit int
}

// Marshal marshales the whole db statement in Json format to include it in the db statement attribute.
func (m *limitedStatementMarshaller) Marshal(command bson.Raw) []attribute.KeyValue {
	b, _ := bson.MarshalExtJSON(command, false, false)
	statement := string(b)
	if m.limit > 0 {
		statement = statement[:m.limit]
	}
	return []attribute.KeyValue{semconv.DBStatement(statement)}
}

// NewDefaultStatementMarshaller implements StatementMarshaller.
// WARNING: This implementation marshals the whole db statement without any sanitization
// or size limit and could lead to leaks of sensitive information and performance issues in the collector.
func NewDefaultStatementMarshaller() StatementMarshaller {
	return &defaultStatementMarshaller{}
}

// defaultStatementMarshaller implements StatementMarshaller.
// WARNING: This implementation marshals the whole db statement without any sanitization
// or size limit and could lead to leaks of sensitive information and performance issues in the collector.
type defaultStatementMarshaller struct {
}

// Marshal marshales the whole db statement in Json format to include it in the db statement attribute.
func (m *defaultStatementMarshaller) Marshal(command bson.Raw) []attribute.KeyValue {
	b, _ := bson.MarshalExtJSON(command, false, false)
	return []attribute.KeyValue{semconv.DBStatement(string(b))}
}
