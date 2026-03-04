package query

import (
	"gorm.io/gorm"
)

// Column 代表数据库中的列名，是一个强类型字符串
// 通过给 Column 增加方法，我们可以实现 model.User.Name.Eq("Tom") 这样的语法
type Column string

// Eq 等于 (Field = Value)
func (c Column) Eq(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " = " + string(col))
		}
		return db.Where(string(c)+" = ?", val)
	}
}

// Neq 不等于 (Field <> Value)
func (c Column) Neq(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " <> " + string(col))
		}
		return db.Where(string(c)+" <> ?", val)
	}
}

// Gt 大于 (Field > Value)
func (c Column) Gt(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " > " + string(col))
		}
		return db.Where(string(c)+" > ?", val)
	}
}

// Gte 大于等于 (Field >= Value)
func (c Column) Gte(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " >= " + string(col))
		}
		return db.Where(string(c)+" >= ?", val)
	}
}

// Lt 小于 (Field < Value)
func (c Column) Lt(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " < " + string(col))
		}
		return db.Where(string(c)+" < ?", val)
	}
}

// Lte 小于等于 (Field <= Value)
func (c Column) Lte(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c) + " <= " + string(col))
		}
		return db.Where(string(c)+" <= ?", val)
	}
}

// Like 模糊查询 (Field LIKE Value)
func (c Column) Like(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c)+" LIKE "+string(col))
		}
		return db.Where(string(c)+" LIKE ?", val)
	}
}

// NotLike 模糊查询否定 (Field NOT LIKE Value)
func (c Column) NotLike(val interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		if col, ok := val.(Column); ok {
			return db.Where(string(c)+" NOT LIKE "+string(col))
		}
		return db.Where(string(c)+" NOT LIKE ?", val)
	}
}

// HasPrefix 前缀匹配 (Field LIKE 'Value%')
func (c Column) HasPrefix(val string) Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c)+" LIKE ?", val+"%")
	}
}

// HasSuffix 后缀匹配 (Field LIKE '%Value')
func (c Column) HasSuffix(val string) Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c)+" LIKE ?", "%"+val)
	}
}

// In 包含 (Field IN Values)
func (c Column) In(vals interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c)+" IN ?", vals)
	}
}

// NotIn 不包含 (Field NOT IN Values)
func (c Column) NotIn(vals interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c)+" NOT IN ?", vals)
	}
}

// IsNull 为空 (Field IS NULL)
func (c Column) IsNull() Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c) + " IS NULL")
	}
}

// IsNotNull 不为空 (Field IS NOT NULL)
func (c Column) IsNotNull() Condition {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(string(c) + " IS NOT NULL")
	}
}

// Between 区间查询 (Field BETWEEN Start AND End)
func (c Column) Between(start, end interface{}) Condition {
	return func(db *gorm.DB) *gorm.DB {
		startStr := "?"
		endStr := "?"
		var args []interface{}

		if col, ok := start.(Column); ok {
			startStr = string(col)
		} else {
			args = append(args, start)
		}

		if col, ok := end.(Column); ok {
			endStr = string(col)
		} else {
			args = append(args, end)
		}

		return db.Where(string(c)+" BETWEEN "+startStr+" AND "+endStr, args...)
	}
}
