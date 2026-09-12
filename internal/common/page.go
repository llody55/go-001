package common

import "gorm.io/gorm"

// PageQuery 统一分页请求参数。
type PageQuery struct {
	PageNum  int `form:"pageNum" json:"pageNum"`
	PageSize int `form:"pageSize" json:"pageSize"`
}

// Normalize 返回 1 基页码与每页条数；非法值回落到默认值。
func (p PageQuery) Normalize() (int, int) {
	page, size := p.PageNum, p.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 500 {
		size = 500
	}
	return page, size
}

// Paginate 分页必须走这个 scope，Service 内不得自行写死 limit/offset。
func Paginate(p PageQuery) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		page, size := p.Normalize()
		return db.Limit(size).Offset((page - 1) * size)
	}
}

// ListResult 统一列表返回结构。
type ListResult struct {
	List  any   `json:"list"`
	Total int64 `json:"total"`
}
