package cali

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"metrobase/internal/response"
)

// PlanController 年度检定计划与到期提醒接口。
type PlanController struct {
	svc *PlanService
}

func NewPlanController(svc *PlanService) *PlanController { return &PlanController{svc: svc} }

// Register 在受保护的 /api 路由组下注册计划接口。
func (ctl *PlanController) Register(rg *gin.RouterGroup) {
	plan := rg.Group("/calibration-plans")
	plan.POST("/generate", ctl.generate)
	plan.GET("", ctl.list)

	notice := rg.Group("/calibration-notice")
	notice.POST("/enable", ctl.enable)
	notice.POST("/scan", ctl.scan)
}

func (ctl *PlanController) generate(c *gin.Context) {
	var in GeneratePlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数不正确")
		return
	}
	if in.Year == 0 {
		in.Year = time.Now().Year()
	}
	rows, err := ctl.svc.Generate(c.Request.Context(), in)
	if err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, gin.H{"generated": len(rows), "list": rows})
}

func (ctl *PlanController) list(c *gin.Context) {
	var q PlanListQuery
	q.Year, _ = strconv.Atoi(c.Query("year"))
	q.DueState = c.Query("dueState")
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

func (ctl *PlanController) enable(c *gin.Context) {
	if err := ctl.svc.EnableNoticeJob(c.Request.Context(), "0 8 * * *"); err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, gin.H{"enabled": 1})
}

func (ctl *PlanController) scan(c *gin.Context) {
	ctl.svc.ScanOverdue(c.Request.Context())
	response.OK(c, gin.H{"scanned": 1})
}
