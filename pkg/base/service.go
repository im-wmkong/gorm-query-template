package base

import (
	"context"
	"gorm-query-template/pkg/db"
)

// Service 定义了通用的 Service 接口
type Service interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

var _ Service = (*BaseService)(nil)

// BaseService 实现了通用的业务逻辑
type BaseService struct {
	tm db.TransactionManager
}

// NewService 创建一个新的 BaseService
func NewService(tm db.TransactionManager) *BaseService {
	return &BaseService{tm: tm}
}

func (s *BaseService) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.tm.Transaction(ctx, fn)
}
