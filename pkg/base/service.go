package base

import (
	"context"

	"gorm-query-template/pkg/query"
)

// Service 定义了通用的 Service 接口
type Service[T any] interface {
	First(ctx context.Context, qb *query.Builder) (*T, error)
	Find(ctx context.Context, qb *query.Builder) ([]*T, error)
	Count(ctx context.Context, qb *query.Builder) (int64, error)
	Create(ctx context.Context, entity *T) error
	Save(ctx context.Context, entity *T) error
	Update(ctx context.Context, column string, value interface{}) error
	Updates(ctx context.Context, values interface{}) error
	Delete(ctx context.Context, qb *query.Builder) error
}

var _ Service[any] = (*BaseService[any])(nil)

// BaseService 实现了通用的业务逻辑
type BaseService[T any] struct {
	Repo Repository[T]
}

// NewService 创建一个新的 BaseService
func NewService[T any](repo Repository[T]) *BaseService[T] {
	return &BaseService[T]{Repo: repo}
}

func (s *BaseService[T]) First(ctx context.Context, qb *query.Builder) (*T, error) {
	return s.Repo.First(ctx, qb)
}

func (s *BaseService[T]) Find(ctx context.Context, qb *query.Builder) ([]*T, error) {
	return s.Repo.Find(ctx, qb)
}

func (s *BaseService[T]) Count(ctx context.Context, qb *query.Builder) (int64, error) {
	return s.Repo.Count(ctx, qb)
}

func (s *BaseService[T]) Create(ctx context.Context, entity *T) error {
	return s.Repo.Create(ctx, entity)
}

func (s *BaseService[T]) Save(ctx context.Context, entity *T) error {
	return s.Repo.Save(ctx, entity)
}

func (s *BaseService[T]) Update(ctx context.Context, column string, value interface{}) error {
	return s.Repo.Update(ctx, column, value)
}

func (s *BaseService[T]) Updates(ctx context.Context, values interface{}) error {
	return s.Repo.Updates(ctx, values)
}

func (s *BaseService[T]) Delete(ctx context.Context, qb *query.Builder) error {
	return s.Repo.Delete(ctx, qb)
}
