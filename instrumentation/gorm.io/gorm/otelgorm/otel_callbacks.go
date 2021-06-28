package otelgorm

import (
	"gorm.io/gorm"
)

func (p otelPlugin) beforeCreate(db *gorm.DB) {
	p.injectBefore(db, _createOp)
}

func (p otelPlugin) after(db *gorm.DB) {
	p.extractAfter(db)
}

func (p otelPlugin) beforeUpdate(db *gorm.DB) {
	p.injectBefore(db, _updateOp)
}

func (p otelPlugin) beforeQuery(db *gorm.DB) {
	p.injectBefore(db, _queryOp)
}

func (p otelPlugin) beforeDelete(db *gorm.DB) {
	p.injectBefore(db, _deleteOp)
}

func (p otelPlugin) beforeRow(db *gorm.DB) {
	p.injectBefore(db, _rowOp)
}

func (p otelPlugin) beforeRaw(db *gorm.DB) {
	p.injectBefore(db, _rawOp)
}
