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

// 催检任务在 t_sys_job 中的函数标识，调度器据此从库里恢复。
const noticeFuncKey = "calibration_due_notice"

// earlyFinishToleranceDays 提前完工容差（天）：校准日期早于计划日期不超过该天数时，
// 发证即核销该计划行。取值小于常见最短检定周期（1 个月），不会跨行核销下一期。
const earlyFinishToleranceDays = 30

// planMatchEnd 计划核销窗口右端：校准日期 + 提前完工容差，格式 yyyy-MM-dd。
// 校准日期非法时原样返回，查询匹配不到任何行，由后续日期校验报错。
func planMatchEnd(calibDate string) string {
	t, err := time.ParseInLocation("2006-01-02", calibDate, time.Local)
	if err != nil {
		return calibDate
	}
	return t.AddDate(0, 0, earlyFinishToleranceDays).Format("2006-01-02")
}

// planMatchStart 计划核销窗口左端：校准日期 - 一个校准周期，格式 yyyy-MM-dd。
// 核销只允许落在本次校准对应的周期内，避免晚发证时错销更早的往期计划行。
func planMatchStart(calibDate string, cycleMonths int) string {
	t, err := time.ParseInLocation("2006-01-02", calibDate, time.Local)
	if err != nil {
		return calibDate
	}
	return t.AddDate(0, -cycleMonths, 0).Format("2006-01-02")
}

func nearestPlan(plans []TCaliPlan, calib time.Time) TCaliPlan {
	if len(plans) == 0 {
		return TCaliPlan{}
	}
	selected := plans[0]
	selectedDays := daysBetween(calib, selected.PlanDate)
	for _, plan := range plans[1:] {
		days := daysBetween(calib, plan.PlanDate)
		if days < selectedDays || (days == selectedDays && plan.PlanDate > selected.PlanDate) ||
			(days == selectedDays && plan.PlanDate == selected.PlanDate && plan.ID > selected.ID) {
			selected = plan
			selectedDays = days
		}
	}
	return selected
}

func daysBetween(base time.Time, date string) int {
	t, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return int(^uint(0) >> 1)
	}
	days := int(base.Sub(t).Hours() / 24)
	if days < 0 {
		return -days
	}
	return days
}

// PlanService 年度检定计划编制、到期看板与提醒。
type PlanService struct {
	db    *gorm.DB
	sched *cronjob.Scheduler
}

func NewPlanService(gdb *gorm.DB, sched *cronjob.Scheduler) *PlanService {
	s := &PlanService{db: gdb, sched: sched}
	if sched != nil {
		// 任务体只注册一次；是否启用、几点执行，以 t_sys_job 里的持久化配置为准。
		sched.RegisterFunc(noticeFuncKey, func() {
			_, _ = s.ScanOverdue(context.Background())
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
	DueState string `form:"dueState" json:"dueState"` // overdue=已超期 soon=30天内到期 done=已完成
	Page     common.PageQuery
}

// Generate 按器具档案的上次校准日期与检定周期，排出指定年度内的应校准日期。
// 同一 (器具, 计划日期) 不重复建行，可对同一年度重复执行补排。
func (s *PlanService) Generate(ctx context.Context, in GeneratePlanInput) ([]TCaliPlan, error) {
	if in.Year < 2000 || in.Year > 2200 {
		return nil, common.NewBizError("计划年度不正确")
	}
	var devs []TCaliDevice
	// 停用器具不纳入年度计划。
	if err := s.db.WithContext(ctx).Clauses(db.Read()).
		Where("status = 0 and last_calib_date <> '' and cycle_months > 0").
		Find(&devs).Error; err != nil {
		return nil, err
	}

	yearStart := time.Date(in.Year, 1, 1, 0, 0, 0, 0, time.Local)
	yearEnd := time.Date(in.Year+1, 1, 1, 0, 0, 0, 0, time.Local)

	// 已排计划用于幂等去重，同时作为计划编号续号依据。
	var existed []TCaliPlan
	if err := s.db.WithContext(ctx).Clauses(db.Read()).
		Select("device_id", "plan_date").
		Where("year = ?", in.Year).Find(&existed).Error; err != nil {
		return nil, err
	}
	existSet := make(map[string]struct{}, len(existed))
	for _, p := range existed {
		existSet[planDedupKey(p.DeviceID, p.PlanDate)] = struct{}{}
	}

	var rows []TCaliPlan
	for _, dev := range devs {
		last, err := time.ParseInLocation("2006-01-02", dev.LastCalibDate, time.Local)
		if err != nil {
			continue
		}
		due := last.AddDate(0, dev.CycleMonths, 0)
		for due.Before(yearEnd) {
			if !due.Before(yearStart) {
				date := due.Format("2006-01-02")
				if _, ok := existSet[planDedupKey(dev.ID, date)]; !ok {
					rows = append(rows, TCaliPlan{
						Year:       in.Year,
						DeviceID:   dev.ID,
						DeviceNo:   dev.DeviceNo,
						DeviceName: dev.DeviceName,
						PlanDate:   date,
						Status:     0,
					})
					existSet[planDedupKey(dev.ID, date)] = struct{}{}
				}
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
		rows[i].PlanNo = fmt.Sprintf("RP-%d-%03d", in.Year, len(existed)+i+1)
	}
	if len(rows) > 0 {
		if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func planDedupKey(deviceID uint64, date string) string {
	return fmt.Sprintf("%d@%s", deviceID, date)
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
	case "overdue": // 已超期：未完成且计划日期早于今天
		q = q.Where("status = 0 and plan_date < ?", today)
	case "soon": // 30 天内到期：含今天起 30 天
		q = q.Where("status = 0 and plan_date >= ? and plan_date <= ?", today, soonEnd)
	case "done": // 已完成
		q = q.Where("status = 1")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := q.Scopes(common.Paginate(query.Page)).
		Order("plan_date asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return &common.ListResult{List: rows, Total: total}, nil
}

// EnableNoticeJob 启用每日催检任务：配置（含 cron 表达式）持久化到 t_sys_job
// 并立即加入调度；进程重启后由调度器从库里自动恢复，重复开启会被拒绝。
func (s *PlanService) EnableNoticeJob(ctx context.Context, cronExpr string) error {
	if cronExpr == "" {
		return common.NewBizError("执行时间表达式不能为空")
	}
	if s.sched == nil {
		return common.NewBizError("定时任务调度器未就绪")
	}
	return s.sched.AddJob(cronjob.TSysJob{
		JobName:  "每日到期催检",
		FuncKey:  noticeFuncKey,
		CronExpr: cronExpr,
		Status:   0,
	})
}

// ScanOverdue 每日催检：扫描到期（含当天）仍未完成的计划行，刷新提醒时间。
// 每天执行都会覆盖 notice_time，因此它记录的是最近一次催检时间。
func (s *PlanService) ScanOverdue(ctx context.Context) (int64, error) {
	today := time.Now().Format("2006-01-02")
	tx := s.db.WithContext(ctx).Model(&TCaliPlan{}).
		Where("status = 0 and plan_date <= ?", today).
		Update("notice_time", time.Now())
	return tx.RowsAffected, tx.Error
}

// AfterIssue 发证后的事务内联动：核销对应的待安排计划行，并按校准日期滚动档案
// 检定日期。任一步失败随发证事务整体回滚。
func (s *PlanService) AfterIssue(ctx context.Context, tx *gorm.DB, rec *TCaliRecord) error {
	if rec == nil {
		return nil
	}
	calib, err := time.ParseInLocation("2006-01-02", rec.CalibDate, time.Local)
	if err != nil {
		return common.NewBizError("校准日期格式不正确，应为 yyyy-MM-dd")
	}

	var dev TCaliDevice
	if err := tx.WithContext(ctx).First(&dev, rec.DeviceID).Error; err != nil {
		return err
	}

	// 一次发证只核销本次校准对应周期内的一行计划：计划日期不早于校准日期前一个周期，
	// 且不晚于校准日期 + 提前完工容差。这样往期漏检行不会被本次发证冲销，当前周期的
	// 计划行也不会继续挂待安排。
	var plans []TCaliPlan
	err = tx.WithContext(ctx).
		Where("device_id = ? and status = 0 and plan_date >= ? and plan_date <= ?",
			rec.DeviceID, planMatchStart(rec.CalibDate, dev.CycleMonths), planMatchEnd(rec.CalibDate)).
		Order("plan_date asc, id asc").Find(&plans).Error
	if err != nil {
		return err
	}
	plan := nearestPlan(plans, calib)
	if plan.ID != 0 {
		if err := tx.WithContext(ctx).Model(&TCaliPlan{}).Where("id = ?", plan.ID).
			Updates(map[string]any{"status": 1, "record_id": rec.ID}).Error; err != nil {
			return err
		}
	}

	return tx.WithContext(ctx).Model(&TCaliDevice{}).Where("id = ?", dev.ID).
		Updates(map[string]any{
			"last_calib_date": rec.CalibDate,
			"next_calib_date": calib.AddDate(0, dev.CycleMonths, 0).Format("2006-01-02"),
		}).Error
}
