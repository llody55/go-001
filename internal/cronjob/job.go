// Package cronjob 定时任务：任务配置持久化在 t_sys_job，进程启动时从库中加载恢复，
// 避免只使用进程内定时器在重启后任务丢失、与任务管理入口脱节。
package cronjob

import (
	"sync"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"metrobase/internal/common"
)

// TSysJob 定时任务配置表。
type TSysJob struct {
	common.BaseModel
	JobName  string `gorm:"column:job_name;size:128" json:"jobName"`
	FuncKey  string `gorm:"column:func_key;size:64" json:"funcKey"`
	CronExpr string `gorm:"column:cron_expr;size:64" json:"cronExpr"`
	Status   int    `gorm:"column:status" json:"status"` // 0=启用 1=暂停
}

func (TSysJob) TableName() string { return "t_sys_job" }

// Scheduler 从 DB 恢复任务的调度器。
type Scheduler struct {
	db    *gorm.DB
	cron  *cron.Cron
	funcs map[string]func()
	mu    sync.Mutex
}

func NewScheduler(gdb *gorm.DB) *Scheduler {
	return &Scheduler{
		db:    gdb,
		cron:  cron.New(),
		funcs: map[string]func(){},
	}
}

// RegisterFunc 注册一个可被任务表引用的函数实现。
func (s *Scheduler) RegisterFunc(key string, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.funcs[key] = f
}

// Start 从 t_sys_job 加载全部启用任务并启动调度。
func (s *Scheduler) Start() error {
	var jobs []TSysJob
	if err := s.db.Find(&jobs).Error; err != nil {
		return err
	}
	for _, job := range jobs {
		if job.Status != 0 {
			continue
		}
		if err := s.scheduleEntry(job); err != nil {
			return err
		}
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() { s.cron.Stop() }

// AddJob 配置落库后再加入调度；同一 func_key 只允许一条启用配置，重复开启返回业务错误。
func (s *Scheduler) AddJob(job TSysJob) error {
	s.mu.Lock()
	f, ok := s.funcs[job.FuncKey]
	s.mu.Unlock()
	if !ok {
		return common.NewBizError("未注册的任务类型：" + job.FuncKey)
	}
	if _, err := cron.ParseStandard(job.CronExpr); err != nil {
		return common.NewBizError("cron 表达式不正确：" + job.CronExpr)
	}
	var dup int64
	if err := s.db.Model(&TSysJob{}).
		Where("func_key = ? and status = 0", job.FuncKey).Count(&dup).Error; err != nil {
		return err
	}
	if dup > 0 {
		return common.NewBizError("该任务已启用，请勿重复开启")
	}
	if err := s.db.Create(&job).Error; err != nil {
		return err
	}
	if _, err := s.cron.AddFunc(job.CronExpr, f); err != nil {
		return common.NewBizError("cron 表达式不正确：" + job.CronExpr)
	}
	return nil
}

// scheduleEntry 只加入调度、不落库（启动恢复时用，避免把库里已有的任务再插一遍）。
func (s *Scheduler) scheduleEntry(job TSysJob) error {
	s.mu.Lock()
	f, ok := s.funcs[job.FuncKey]
	s.mu.Unlock()
	if !ok {
		return common.NewBizError("未注册的任务类型：" + job.FuncKey)
	}
	if _, err := s.cron.AddFunc(job.CronExpr, f); err != nil {
		return common.NewBizError("cron 表达式不正确：" + job.CronExpr)
	}
	return nil
}
