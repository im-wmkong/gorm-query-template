package repository

import (
	"gorm-query-template/internal/model"
	"gorm-query-template/pkg/base"
	"gorm-query-template/pkg/db"
)

// UserRepository 接口继承了通用的 Repository 接口
// 你可以在这里添加针对 User 的特定方法
type UserRepository interface {
	base.Repository[model.User]
}

// userRepository 实现继承自 BaseRepository
type userRepository struct {
	*base.BaseRepository[model.User]
}

// NewUserRepository 创建一个新的 user repository
func NewUserRepository(connector db.Connector) UserRepository {
	return &userRepository{
		BaseRepository: base.NewRepository[model.User](connector),
	}
}
