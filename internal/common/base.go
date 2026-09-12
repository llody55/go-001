package common

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 所有业务表的公共审计列，列名由脚手架统一约定（对应表必须有这些列）。
// create_by/create_time/update_by/update_time 由 store 层回调统一填充，
// 业务代码不应手动给这些列赋值。
type BaseModel struct {
	ID         uint64         `gorm:"primaryKey;column:id" json:"id"`
	CreateBy   string         `gorm:"column:create_by" json:"createBy"`
	CreateTime *time.Time     `gorm:"column:create_time" json:"createTime"`
	UpdateBy   string         `gorm:"column:update_by" json:"updateBy"`
	UpdateTime *time.Time     `gorm:"column:update_time" json:"updateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:del_flag;index" json:"-"`
}
