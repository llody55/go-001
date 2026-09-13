package cali

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"metrobase/internal/common"
	"metrobase/internal/cronjob"
	"metrobase/internal/db"
)

// PlanService 年度检定计划编制、到期看板与提醒。
type PlanService struct {
	db    *gorm.DB
	sched *cronjob.Scheduler
}

func NewPlanService(gdb *gorm.DB, sched *cronjob.Scheduler) *PlanService {
	s := &PlanService{db: gdb, sched: sched}
	if sched != nil {
		sched.RegisterFunc("calibration_due_notice", func() {
			s.ScanOverdue(context.Background())
		})
	}
	return s
}

type GeneratePlanInput struct {
	Year int `json:"year"`
}

type PlanListQuery struct {
	Year     int    `form:"year" json:"year"`
	Status   *int   `form:"status" json:"status"`
	DueState string `form:"dueState" json:"dueState"` // overdue=已超期 soon=30天内到期
	Page     common.PageQuery
}

// Generate 按器具档案的周期与上次校准日期，排出指定年度内的应校准日期。
func (s *PlanService) Generate(ctx context.Context, in GeneratePlanInput) ([]TCaliPlan, error) {
	if in.Year < 2000 || in.Year > 2200 {
		return nil, common.NewBizError("计划年度不正确")
	}
	var devs []TCaliDevice
	if err := s.db.WithContext(ctx).Find(&devs).Error; err != nil {
		return nil, err
	}

	yearStart := time.Date(in.Year, 1, 1, 0, 0, 0, 0, time.Local)
	yearEnd := time.Date(in.Year+1, 1, 1, 0, 0, 0, 0, time.Local)

	var rows []TCaliPlan
	for _, dev := range devs {
		if dev.LastCalibDate == "" || dev.CycleMonths <= 0 {
			continue
		}
		last, err := time.ParseInLocation("2006-01-02", dev.LastCalibDate, time.Local)
		if err != nil {
			continue
		}
		due := last.AddDate(0, dev.CycleMonths, 0)
		for due.Before(yearEnd) {
			if due.After(yearStart) {
				rows = append(rows, TCaliPlan{
					Year:       in.Year,
					DeviceID:   dev.ID,
					DeviceNo:   dev.DeviceNo,
					DeviceName: dev.DeviceName,
					PlanDate:   due.Format("2006-01-02"),
					Status:     0,
				})
			}
			due = due.AddDate(0, dev.CycleMonths, 0)
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PlanDate == rows[j].PlanDate {
			return rows[i].DeviceID < rows[j].DeviceID
		}
		return rows[i].PlanDate < rows[j].PlanDate
	})
	for i := range rows {
		rows[i].PlanNo = fmt.Sprintf("RP-%d-%03d", in.Year, i+1)
	}
	if len(rows) > 0 {
		if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// List 到期看板：按年度、状态、到期情况查询，分页返回。
func (s *PlanService) List(ctx context.Context, query PlanListQuery) (*common.ListResult, error) {
	var rows []TCaliPlan
	var total int64

	q := s.db.WithContext(ctx).Clauses(db.Read()).Model(&TCaliPlan{})
	if query.Year > 0 {
		q = q.Where("year = ?", query.Year)
	}
	if query.Status != nil {
		q = q.Where("status = ?", *query.Status)
	}
	today := time.Now().Format("2006-01-02")
	soonEnd := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	switch query.DueState {
	case "overdue":
		q = q.Where("status = 0 and plan_date < ?", today)
	case "soon":
		q = q.Where("status = 0 and plan_date >= ? and plan_date <= ?", today, soonEnd)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := q.Scopes(common.Paginate(query.Page)).
		Order("plan_date asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return &common.ListResult{List: rows, Total: total}, nil
}

// EnableNoticeJob 启用每日到期提醒任务。
func (s *PlanService) EnableNoticeJob(ctx context.Context, cronExpr string) error {
	if cronExpr == "" {
		return common.NewBizError("执行时间表达式不能为空")
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.ScanOverdue(context.Background())
		}
	}()
	return nil
}

// ScanOverdue 扫描超期未完成的计划行，记一次提醒时间。
func (s *PlanService) ScanOverdue(ctx context.Context) {
	today := time.Now().Format("2006-01-02")
	s.db.WithContext(ctx).Model(&TCaliPlan{}).
		Where("status = 0 and plan_date < ? and notice_time is null", today).
		Update("notice_time", time.Now())
}

// AfterCalibration 一条校准记录落库后，联动核销计划并滚动档案下次到期日。
func (s *PlanService) AfterCalibration(ctx context.Context, rec *TCaliRecord) {
	if rec == nil || rec.CalibDate == "" || len(rec.CalibDate) < 4 {
		return
	}
	year := 0
	for i := 0; i < 4; i++ {
		year = year*10 + int(rec.CalibDate[i]-'0')
	}
	s.db.WithContext(ctx).Model(&TCaliPlan{}).
		Where("device_id = ? and year = ? and status = 0", rec.DeviceID, year).
		Updates(map[string]any{"status": 1, "record_id": rec.ID})

	var dev TCaliDevice
	if err := s.db.WithContext(ctx).First(&dev, rec.DeviceID).Error; err != nil {
		return
	}
	now := time.Now()
	s.db.WithContext(ctx).Model(&TCaliDevice{}).Where("id = ?", dev.ID).
		Updates(map[string]any{
			"last_calib_date": now.Format("2006-01-02"),
			"next_calib_date": now.AddDate(0, dev.CycleMonths, 0).Format("2006-01-02"),
		})
}
