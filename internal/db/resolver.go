package db

import (
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"
)

// Read 标注本次查询走从库（读连接）。读接口必须显式带上：
//
//	db.Clauses(db.Read()).Where(...).Find(...)
//
//	不带则走主库。
func Read() clause.Expression { return dbresolver.Read }
