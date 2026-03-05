# 🛡️ GORM Query Template

基于 GORM 构建的 Go 语言 Web 数据访问层（DAL）现代化脚手架。

本项目旨在解决传统 Go Web 开发中的三大痛点：**Repository 层充斥大量重复的 CRUD 代码**、**Service 层难以灵活定制复杂查询**，以及**字符串硬编码导致的极高维护成本**。通过引入 Go 1.18+ 泛型与 AST 代码生成技术，为您提供一个类型安全、零硬编码、高度可扩展的持久层解决方案。

## ✨ 核心特性

* **🪶 极简的 Repository 层 (基于泛型)**
  借助 `base.Repository[T]`，你的业务 Repo 默认集成 Create, Delete, Update, First, Find, Count 等全套标准操作。告别机械化的样板代码，让 Repository 真正回归“数据访问”的本质。

* **🛠️ Service 层掌控查询 (链式调用)**
  释放 Service 层的自由度。通过 `pkg/query` 提供的查询构建器，直接在 Service 层以流畅的链式 API 组装复杂查询条件，无需再为每一种查询组合在 Repo 层新增方法。

* **🚫 零硬编码 (类型安全的字段映射)**
  告别 `Where("user_name = ?", name)` 这种容易拼写错误的魔法字符串。搭配内置的 `cmd/gen-props` 生成器，自动提取 Model 字段并生成强类型常量（如 `UserProps.UserName.Eq()`），在编译期即可拦截拼写错误。

* **📦 开箱即用**
  提供标准的项目分层结构（Service -> Repository -> Model），支持无缝接入现有项目。

## 📂 目录结构

```text
.
├── cmd/
│   └── gen-props/      # 核心逻辑：基于 AST 的字段常量生成工具
├── internal/
│   ├── model/          # 实体层：定义数据库表映射结构 (Domain Models)
│   ├── repository/     # 数据层：基于泛型的接口与具体实现
│   └── service/        # 业务层：在此层利用 query 库组装并执行查询
├── pkg/
│   ├── base/           # 基础设施：泛型基类 (BaseModel, BaseRepo, BaseService)
│   └── query/          # 核心引擎：无反射的强类型查询构建器
├── test/               # 测试套件：单元测试与全链路集成测试
├── go.mod
└── README.md
```

## 🚀 快速开始

### 1. 定义数据模型 (Model)

在 `internal/model` 目录下定义你的业务实体，嵌入 `base.Model` 获取通用字段（如 ID、创建时间等）。
**关键：** 添加 `//go:generate` 指令，绑定代码生成器。

```go
package model

import "gorm-query-template/pkg/base"

//go:generate go run ../../cmd/gen-props -type User -output user_gen.go

type User struct {
    base.Model
    UserName string `gorm:"column:user_name;size:255;not null"`
    Email    string `gorm:"column:email;size:255;unique"`
    Age      int    `gorm:"column:age"`
    Status   int    `gorm:"column:status;default:1"`
}

func (User) TableName() string {
    return "users"
}
```

### 2. 生成强类型字段常量

执行生成命令。此操作将自动扫描结构体标签，生成 `user_gen.go` 以及全局可用的 `UserProps` 对象。

```bash
cd internal/model
go generate
```

### 3. 初始化 Repository

继承泛型接口 `base.Repository[T]` 和实现类 `base.BaseRepository[T]`。**零代码**即可获得完整的 CRUD 能力。

```go
package repository

import (
    "gorm.io/gorm"
    "gorm-query-template/internal/model"
    "gorm-query-template/pkg/base"
)

type UserRepository interface {
    base.Repository[model.User]
}

type userRepository struct {
    *base.BaseRepository[model.User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{
        BaseRepository: base.NewRepository[model.User](db),
    }
}
```

### 4. 在 Service 中优雅地构建查询

在 Service 层继承 `base.BaseService[T]`，结合 `pkg/query` 和刚才生成的 `UserProps`，即可实现类型安全、高可读性的查询逻辑。

```go
package service

import (
    "context"
    "gorm-query-template/internal/model"
    "gorm-query-template/pkg/query"
)

// GetActiveUsers 获取符合条件的活跃用户
func (s *userService) GetActiveUsers(ctx context.Context, minAge int) ([]*model.User, error) {
    // 💡 链式构建查询：安全、直观、无魔法字符串
    q := query.New().
        Where(
            model.UserProps.Status.Eq(1),      // 编译期安全的 status = 1
            model.UserProps.Age.Gte(minAge),   // 编译期安全的 age >= minAge
        ).
        Order(model.UserProps.CreatedAt, true) // 按 CreatedAt DESC 排序

    // 调用底层封装好的 Find 方法执行查询
    return s.Find(ctx, q)
}
```

## 🧪 测试

项目中包含了全面的单元与集成测试用例，帮助您快速验证组件联动效果。

```bash
go test -v ./test/...
```

## 🛠️ 技术栈

* **Language**: Go 1.18+ (依赖泛型特性)
* **ORM**: GORM v1.31+
* **Database**: SQLite (默认演示), 完全兼容 MySQL / PostgreSQL
* **Testing**: Testify

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 协议开源。
