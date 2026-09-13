package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"metrobase/internal/auth"
	"metrobase/internal/cali"
	"metrobase/internal/cronjob"
	"metrobase/internal/response"
	"metrobase/internal/sys"
)

// NewEngine 装配全部 HTTP 路由。新增业务模块在此注册受保护路由组。
func NewEngine(gdb *gorm.DB, jwtSecret string, expireHours int) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), response.Recovery())

	pub := r.Group("/api")
	authSvc := sys.NewAuthService(gdb, jwtSecret, expireHours)
	sysCtl := sys.NewController(authSvc)
	pub.POST("/login", sysCtl.Login)

	protected := r.Group("/api")
	protected.Use(auth.Middleware(jwtSecret))

	// 定时任务从 t_sys_job 恢复（进程重启不丢任务配置）
	scheduler := cronjob.NewScheduler(gdb)

	recordSvc := cali.NewRecordService(gdb)
	planSvc := cali.NewPlanService(gdb, scheduler)
	recordSvc.AfterIssue = planSvc.AfterIssue

	caliCtl := cali.NewController(recordSvc)
	caliCtl.Register(protected)

	planCtl := cali.NewPlanController(planSvc)
	planCtl.Register(protected)

	_ = scheduler.Start()

	return r
}
