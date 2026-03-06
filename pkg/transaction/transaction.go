package transaction

import (
	"context"
	"gorm-query-template/pkg/db"

	"gorm.io/gorm"
)

// Transactioner 定义事务接口
type Transactioner interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
	DB(ctx context.Context) *gorm.DB
}

type Manager struct {
	connector db.Connector
}

func NewManager(connector db.Connector) *Manager {
	return &Manager{connector: connector}
}

func (m *Manager) DB(ctx context.Context) *gorm.DB {
	return m.connector.DB(ctx)
}

// Transaction 执行事务
func (m *Manager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.DB(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := db.WithTx(ctx, tx)
		return fn(txCtx)
	})
}
