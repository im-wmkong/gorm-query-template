package base

import (
	"context"

	"gorm-query-template/pkg/query"

	"gorm.io/gorm"
)

type txKeyType struct{}

var txKey = txKeyType{}

// Repository 定义了通用的仓储接口
// T 是 Model 类型
type Repository[T any] interface {
	DB(ctx context.Context) *gorm.DB
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
	Create(ctx context.Context, entity *T) error
	Save(ctx context.Context, entity *T) error
	Update(ctx context.Context, column string, value interface{}) error
	Updates(ctx context.Context, values interface{}) error
	Delete(ctx context.Context, q *query.Builder) error
	Find(ctx context.Context, q *query.Builder) ([]*T, error)
	First(ctx context.Context, q *query.Builder) (*T, error)
	Count(ctx context.Context, q *query.Builder) (int64, error)
}

var _ Repository[any] = (*BaseRepository[any])(nil)

// BaseRepository 实现了通用的 CRUD 操作
type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewRepository 创建一个新的 BaseRepository
func NewRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{db: db}
}

func (r *BaseRepository[T]) DB(ctx context.Context) *gorm.DB {
	v := ctx.Value(txKey)
	if v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	return r.db.WithContext(ctx)
}

func (r *BaseRepository[T]) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, txKey, tx)
		return fn(ctx)
	})
}

func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *BaseRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *BaseRepository[T]) Update(ctx context.Context, column string, value interface{}) error {
	var entity T
	return r.DB(ctx).Model(entity).Update(column, value).Error
}

func (r *BaseRepository[T]) Updates(ctx context.Context, values interface{}) error {
	var entity T
	return r.DB(ctx).Model(entity).Updates(values).Error
}

func (r *BaseRepository[T]) Delete(ctx context.Context, qb *query.Builder) error {
	var entity T
	db := r.DB(ctx)
	if qb != nil {
		db = qb.Apply(db)
	}
	return db.Delete(&entity).Error
}

func (r *BaseRepository[T]) Find(ctx context.Context, qb *query.Builder) ([]*T, error) {
	var entities []*T
	db := r.DB(ctx)
	if qb != nil {
		db = qb.Apply(db)
	}
	err := db.Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) First(ctx context.Context, qb *query.Builder) (*T, error) {
	var entity T
	db := r.DB(ctx)
	if qb != nil {
		db = qb.Apply(db)
	}
	err := db.First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T]) Count(ctx context.Context, qb *query.Builder) (int64, error) {
	var count int64
	var entity T
	db := r.DB(ctx).Model(&entity)
	if qb != nil {
		db = qb.Apply(db)
	}
	err := db.Count(&count).Error
	return count, err
}
