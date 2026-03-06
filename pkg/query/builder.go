package query

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Condition 定义一个修改 *gorm.DB 对象的函数
// 这是核心抽象，所有的 WHERE 条件最终都转换为此函数
type Condition func(db *gorm.DB) *gorm.DB

// Builder 查询构建器
type Builder struct {
	conditions []Condition
}

// New 创建一个新的查询构建器
func New() *Builder {
	return &Builder{}
}

// Apply 将所有累积的查询条件应用到 gorm.DB 上
func (b *Builder) Apply(db *gorm.DB) *gorm.DB {
	for _, cond := range b.conditions {
		db = cond(db)
	}
	return db
}

// convertArgs 转换参数列表中的 Column 类型为 string
func convertArgs(args []interface{}) []interface{} {
	newArgs := make([]interface{}, len(args))
	for i, arg := range args {
		if col, ok := arg.(Column); ok {
			newArgs[i] = string(col)
		} else {
			newArgs[i] = arg
		}
	}
	return newArgs
}

// Select 指定查询字段
func (b *Builder) Select(query interface{}, args ...interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		// 转换 args 中的 Column 类型
		newArgs := convertArgs(args)

		// 如果 query 是 Column 类型，转换为 string
		if col, ok := query.(Column); ok {
			return db.Select(string(col), newArgs...)
		}
		return db.Select(query, newArgs...)
	})
	return b
}

// Where 接受一个或多个 Condition 并将其添加到构建器中
func (b *Builder) Where(conds ...Condition) *Builder {
	b.conditions = append(b.conditions, conds...)
	return b
}

// Or 添加 OR 条件
func (b *Builder) Or(conds ...Condition) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		// 使用 DryRun 提取条件以避免 db.Or(func) 出现 "unsupported type func" 错误
		tmpDB := db.Session(&gorm.Session{DryRun: true, NewDB: true})
		for _, c := range conds {
			tmpDB = c(tmpDB)
		}

		if whereClause, ok := tmpDB.Statement.Clauses["WHERE"]; ok {
			if where, ok := whereClause.Expression.(clause.Where); ok {
				// 将所有表达式用 AND 组合（Where 的默认行为）
				if len(where.Exprs) > 0 {
					return db.Or(clause.And(where.Exprs...))
				}
			}
		}
		return db
	})
	return b
}

// Group 分组
func (b *Builder) Group(name interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		if col, ok := name.(Column); ok {
			return db.Group(string(col))
		}
		if str, ok := name.(string); ok {
			return db.Group(str)
		}
		return db
	})
	return b
}

// Having 分组后过滤
func (b *Builder) Having(query interface{}, args ...interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		// 转换 args 中的 Column 类型
		newArgs := convertArgs(args)

		// 如果 query 是 Column 类型，转换为 string
		if col, ok := query.(Column); ok {
			return db.Having(string(col), newArgs...)
		}
		return db.Having(query, newArgs...)
	})
	return b
}

// Joins 连接查询
func (b *Builder) Joins(query string, args ...interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		// Joins 的 args 通常是值，不建议转换 Column 为 string (除非用户确实想拼接 SQL)
		// 如果用户想把 Column 当表名/列名拼进去，最好在 query 字符串里自己拼
		return db.Joins(query, args...)
	})
	return b
}

// Distinct 去重
func (b *Builder) Distinct(args ...interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Distinct(convertArgs(args)...)
	})
	return b
}

// Omit 忽略字段
// 使用 ...interface{} 以支持 Column 类型
func (b *Builder) Omit(columns ...interface{}) *Builder {
	var strs []string
	for _, col := range columns {
		if c, ok := col.(Column); ok {
			strs = append(strs, string(c))
		} else if s, ok := col.(string); ok {
			strs = append(strs, s)
		}
	}

	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Omit(strs...)
	})
	return b
}

// Unscoped 忽略软删除等 Scope
func (b *Builder) Unscoped() *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	})
	return b
}

// Order 排序
// 参数可以是 Column 类型，也可以是字符串
func (b *Builder) Order(col interface{}, desc ...bool) *Builder {
	var orderClause interface{}
	isDesc := len(desc) > 0 && desc[0]

	switch v := col.(type) {
	case Column:
		if isDesc {
			orderClause = string(v) + " DESC"
		} else {
			orderClause = string(v)
		}
	case string:
		// 允许直接传 "created_at DESC" 这种 raw string
		orderClause = v
	default:
		return b
	}

	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Order(orderClause)
	})
	return b
}

// Page 分页辅助方法
func (b *Builder) Page(page, pageSize int) *Builder {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Limit(pageSize).Offset(offset)
	})
	return b
}

// Limit 限制数量
func (b *Builder) Limit(limit int) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	})
	return b
}

// Offset 偏移量
func (b *Builder) Offset(offset int) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset)
	})
	return b
}

// Scope 支持 GORM Scopes
func (b *Builder) Scope(funcs ...func(*gorm.DB) *gorm.DB) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Scopes(funcs...)
	})
	return b
}

// Preload 预加载关联
func (b *Builder) Preload(query string, args ...interface{}) *Builder {
	b.conditions = append(b.conditions, func(db *gorm.DB) *gorm.DB {
		return db.Preload(query, convertArgs(args)...)
	})
	return b
}
