package cali

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"metrobase/internal/response"
)

type Controller struct{ svc *RecordService }

func NewController(svc *RecordService) *Controller { return &Controller{svc: svc} }

// Register 在受保护的 /api 路由组下注册校准记录接口。
func (ctl *Controller) Register(rg *gin.RouterGroup) {
	rec := rg.Group("/calibration-records")
	rec.POST("", ctl.create)
	rec.PUT("/:id", ctl.update)
	rec.POST("/:id/issue", ctl.issue)
	rec.GET("", ctl.list)
}

func (ctl *Controller) create(c *gin.Context) {
	var in RecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数不正确")
		return
	}
	rec, err := ctl.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, rec)
}

func (ctl *Controller) update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var in RecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数不正确")
		return
	}
	if err := ctl.svc.Update(c.Request.Context(), id, in); err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, gin.H{"updated": 1})
}

func (ctl *Controller) issue(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var body struct {
		CertNo string `json:"certNo"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := ctl.svc.Issue(c.Request.Context(), id, body.CertNo); err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, gin.H{"issued": 1})
}

func (ctl *Controller) list(c *gin.Context) {
	var q RecordListQuery
	q.RecordNo = c.Query("recordNo")
	if v := c.Query("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Status = &n
		}
	}
	q.Page.PageNum, _ = strconv.Atoi(c.DefaultQuery("pageNum", "1"))
	q.Page.PageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	result, err := ctl.svc.List(c.Request.Context(), q)
	if err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, result)
}
