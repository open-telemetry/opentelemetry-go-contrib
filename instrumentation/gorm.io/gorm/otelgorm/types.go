package otelgorm

// operationName defines a type to wrap the name of each operation name.
type operationName string

// String returns the actual string of operationName.
func (op operationName) String() string {
	return string(op)
}

const (
	_createOp operationName = "create"
	_updateOp operationName = "update"
	_queryOp  operationName = "query"
	_deleteOp operationName = "delete"
	_rowOp    operationName = "row"
	_rawOp    operationName = "raw"
)

// operationStage indicates the timing when the operation happens.
type operationStage string

// Name returns the actual string of operationStage.
func (op operationStage) Name() string {
	return string(op)
}

const (
	_stageBeforeCreate operationStage = "opentelemetry:before_create"
	_stageAfterCreate  operationStage = "opentelemetry:after_create"
	_stageBeforeUpdate operationStage = "opentelemetry:before_update"
	_stageAfterUpdate  operationStage = "opentelemetry:after_update"
	_stageBeforeQuery  operationStage = "opentelemetry:before_query"
	_stageAfterQuery   operationStage = "opentelemetry:after_query"
	_stageBeforeDelete operationStage = "opentelemetry:before_delete"
	_stageAfterDelete  operationStage = "opentelemetry:after_delete"
	_stageBeforeRow    operationStage = "opentelemetry:before_row"
	_stageAfterRow     operationStage = "opentelemetry:after_row"
	_stageBeforeRaw    operationStage = "opentelemetry:before_raw"
	_stageAfterRaw     operationStage = "opentelemetry:after_raw"
)
