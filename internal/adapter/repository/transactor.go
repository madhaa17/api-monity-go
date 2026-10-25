package repository

import (
	"context"

	"monity/internal/core/port"

	"gorm.io/gorm"
)

type txKey struct{}

type GormTransactor struct {
	db *gorm.DB
}

func NewTransactor(db *gorm.DB) port.Transactor {
	return &GormTransactor{db: db}
}

// WithinTransaction runs fn in a transaction. If ctx already carries one, fn joins it instead of nesting.
func (t *GormTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// conn returns the transaction carried by ctx, or db when there is none.
func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
