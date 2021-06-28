package otelgorm

import (
	"strings"

	"gorm.io/gorm"
)

type otelPlugin struct {
	// opt includes options those otelPlugin support.
	opt *options
}

// New constructs a new plugin based opentelemetry. It supports to trace all operations in gorm,
// so if you have already traced your servers, now this plugin will perfect your tracing job.
func New(opts ...applyOption) gorm.Plugin {
	dst := defaultOption()
	for _, apply := range opts {
		apply(dst)
	}

	return otelPlugin{
		//logResult: dst.logResult,
		opt: dst,
	}
}

func (p otelPlugin) Name() string {
	return "opentelemetry"
}

// Initialize registers all needed callbacks
func (p otelPlugin) Initialize(db *gorm.DB) (err error) {
	e := myError{
		errs: make([]string, 0, 12),
	}

	// create
	err = db.Callback().Create().Before("gorm:create").Register(_stageBeforeCreate.Name(), p.beforeCreate)
	e.add(_stageBeforeCreate, err)
	err = db.Callback().Create().After("gorm:create").Register(_stageAfterCreate.Name(), p.after)
	e.add(_stageAfterCreate, err)

	// update
	err = db.Callback().Update().Before("gorm:update").Register(_stageBeforeUpdate.Name(), p.beforeUpdate)
	e.add(_stageBeforeUpdate, err)
	err = db.Callback().Update().After("gorm:update").Register(_stageAfterUpdate.Name(), p.after)
	e.add(_stageAfterUpdate, err)

	// query
	err = db.Callback().Query().Before("gorm:query").Register(_stageBeforeQuery.Name(), p.beforeQuery)
	e.add(_stageBeforeQuery, err)
	err = db.Callback().Query().After("gorm:query").Register(_stageAfterQuery.Name(), p.after)
	e.add(_stageAfterQuery, err)

	// delete
	err = db.Callback().Delete().Before("gorm:delete").Register(_stageBeforeDelete.Name(), p.beforeDelete)
	e.add(_stageBeforeDelete, err)
	err = db.Callback().Delete().After("gorm:delete").Register(_stageAfterDelete.Name(), p.after)
	e.add(_stageAfterDelete, err)

	// row
	err = db.Callback().Row().Before("gorm:row").Register(_stageBeforeRow.Name(), p.beforeRow)
	e.add(_stageBeforeRow, err)
	err = db.Callback().Row().After("gorm:row").Register(_stageAfterRow.Name(), p.after)
	e.add(_stageAfterRow, err)

	// raw
	err = db.Callback().Raw().Before("gorm:raw").Register(_stageBeforeRaw.Name(), p.beforeRaw)
	e.add(_stageBeforeRaw, err)
	err = db.Callback().Raw().After("gorm:raw").Register(_stageAfterRaw.Name(), p.after)
	e.add(_stageAfterRaw, err)

	return e.toError()
}

type myError struct {
	errs []string
}

func (e *myError) add(stage operationStage, err error) {
	if err == nil {
		return
	}

	e.errs = append(e.errs, "stage="+stage.Name()+":"+err.Error())
}

func (e myError) toError() error {
	if len(e.errs) == 0 {
		return nil
	}

	return e
}

func (e myError) Error() string {
	return strings.Join(e.errs, ";")
}
