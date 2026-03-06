package db

import (
	"context"

	"gorm.io/gorm"
)

type txKeyType struct{}

var txKey = txKeyType{}

// Connector 定义了获取数据库连接的接口
type Connector interface {
	DB(ctx context.Context) *gorm.DB
}

type Client struct {
	db *gorm.DB
}

func NewClient(db *gorm.DB) *Client {
	return &Client{db: db}
}

func (c *Client) DB(ctx context.Context) *gorm.DB {
	v := ctx.Value(txKey)
	if v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	return c.db.WithContext(ctx)
}

// WithTx 将事务 DB 注入 context
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}
