package cali

import (
	"time"

	"metrobase/internal/common"
)

// TCaliDevice 计量器具档案，对应表 t_cali_device。
type TCaliDevice struct {
	common.BaseModel
	DeviceNo      string  `gorm:"column:device_no" json:"deviceNo"`           // 器具编号
	DeviceName    string  `gorm:"column:device_name" json:"deviceName"`       // 器具名称
	ModelSpec     string  `gorm:"column:model_spec" json:"modelSpec"`         // 规格型号
	FullScale     float64 `gorm:"column:full_scale" json:"fullScale"`         // 测量范围上限（量程上限）
	AccuracyClass float64 `gorm:"column:accuracy_class" json:"accuracyClass"` // 准确度等级：允许的引用误差（百分数）
	UseDept       string  `gorm:"column:use_dept" json:"useDept"`             // 使用部门
	Keeper        string  `gorm:"column:keeper" json:"keeper"`                // 保管人
	Status        int     `gorm:"column:status" json:"status"`                // 0=正常 1=停用
	LastCalibDate string  `gorm:"column:last_calib_date" json:"lastCalibDate"`
	CycleMonths   int     `gorm:"column:cycle_months" json:"cycleMonths"`
	NextCalibDate string  `gorm:"column:next_calib_date" json:"nextCalibDate"`
}

func (TCaliDevice) TableName() string { return "t_cali_device" }

// TCaliRecord 校准/检定记录（t_cali_record）。
type TCaliRecord struct {
	common.BaseModel
	RecordNo       string  `gorm:"column:record_no" json:"recordNo"`
	DeviceID       uint64  `gorm:"column:device_id" json:"deviceId"`
	DeviceNo       string  `gorm:"column:device_no" json:"deviceNo"` // 冗余档案字段，以档案为准
	DeviceName     string  `gorm:"column:device_name" json:"deviceName"`
	CalibDate      string  `gorm:"column:calib_date" json:"calibDate"`           // 校准日期（自然日 yyyy-MM-dd）
	StandardValue  float64 `gorm:"column:standard_value" json:"standardValue"`   // 标准值
	IndicatedValue float64 `gorm:"column:indicated_value" json:"indicatedValue"` // 示值
	ReferenceError float64 `gorm:"column:reference_error" json:"referenceError"` // 最大引用误差（百分数，两位小数）
	Result         int     `gorm:"column:result" json:"result"`                  // 0=合格 1=不合格
	CertNo         string  `gorm:"column:cert_no" json:"certNo"`                 // 证书编号（发证后回填）
	Status         int     `gorm:"column:status" json:"status"`                  // 0=已登记 1=已发证 2=作废
	Remark         string  `gorm:"column:remark" json:"remark"`
}

func (TCaliRecord) TableName() string { return "t_cali_record" }

// TCaliPlan 年度检定计划行（t_cali_plan）：一台器具一个应校准日期一行。
type TCaliPlan struct {
	common.BaseModel
	PlanNo     string     `gorm:"column:plan_no" json:"planNo"`
	Year       int        `gorm:"column:year" json:"year"`
	DeviceID   uint64     `gorm:"column:device_id" json:"deviceId"`
	DeviceNo   string     `gorm:"column:device_no" json:"deviceNo"` // 冗余档案字段，以档案为准
	DeviceName string     `gorm:"column:device_name" json:"deviceName"`
	PlanDate   string     `gorm:"column:plan_date" json:"planDate"` // 计划应校准日期（自然日 yyyy-MM-dd）
	Status     int        `gorm:"column:status" json:"status"`      // 0=待安排 1=已完成
	RecordID   *uint64    `gorm:"column:record_id" json:"recordId"`
	NoticeTime *time.Time `gorm:"column:notice_time" json:"noticeTime"`
}

func (TCaliPlan) TableName() string { return "t_cali_plan" }
