package sys

import "metrobase/internal/common"

// TSysUser 系统用户，对应表 t_sys_user。
type TSysUser struct {
	common.BaseModel
	Username string `gorm:"column:username;size:64;uniqueIndex" json:"username"`
	Password string `gorm:"column:password;size:64" json:"-"`
	RealName string `gorm:"column:real_name;size:64" json:"realName"`
	Status   int    `gorm:"column:status" json:"status"` // 0=正常 1=停用
}

func (TSysUser) TableName() string { return "t_sys_user" }
