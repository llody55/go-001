package cali

import (
	"context"
	"math"

	"gorm.io/gorm"

	"metrobase/internal/common"
)

// RecordService 校准记录登记。
type RecordService struct {
	db *gorm.DB
}

func NewRecordService(db *gorm.DB) *RecordService { return &RecordService{db: db} }

type RecordInput struct {
	RecordNo       string  `json:"recordNo"`
	DeviceID       uint64  `json:"deviceId"`
	DeviceNo       string  `json:"deviceNo"`
	DeviceName     string  `json:"deviceName"`
	CalibDate      string  `json:"calibDate"`
	StandardValue  float64 `json:"standardValue"`
	IndicatedValue float64 `json:"indicatedValue"`
	Operator       string  `json:"operator"`
	Remark         string  `json:"remark"`
}

// Create 登记一条校准记录并给出合格结论。
func (s *RecordService) Create(ctx context.Context, in RecordInput) (*TCaliRecord, error) {
	rec := TCaliRecord{
		RecordNo:       in.RecordNo,
		DeviceID:       in.DeviceID,
		DeviceNo:       in.DeviceNo,
		DeviceName:     in.DeviceName,
		CalibDate:      in.CalibDate,
		StandardValue:  in.StandardValue,
		IndicatedValue: in.IndicatedValue,
		Status:         0,
		Remark:         in.Remark,
	}

	// 计算引用误差与合格判定
	var dev TCaliDevice
	s.db.First(&dev, in.DeviceID)
	rate := 0.0
	if in.StandardValue != 0 {
		rate = math.Abs(in.IndicatedValue-in.StandardValue) / in.StandardValue * 100
	}
	rec.ReferenceError = math.Round(rate*100) / 100
	if rate < dev.AccuracyClass {
		rec.Result = 0
	} else {
		rec.Result = 1
	}

	rec.CreateBy = in.Operator

	if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// Update 编辑校准记录。
func (s *RecordService) Update(ctx context.Context, id uint64, in RecordInput) error {
	updates := map[string]any{
		"record_no":       in.RecordNo,
		"device_id":       in.DeviceID,
		"device_no":       in.DeviceNo,
		"device_name":     in.DeviceName,
		"calib_date":      in.CalibDate,
		"standard_value":  in.StandardValue,
		"indicated_value": in.IndicatedValue,
		"remark":          in.Remark,
	}
	return s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("id = ?", id).Updates(updates).Error
}

// Issue 发证：回填证书编号。
func (s *RecordService) Issue(ctx context.Context, id uint64, certNo string) error {
	return s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{"cert_no": certNo, "status": 1}).Error
}

// List 列表查询：记录编号模糊 + 状态筛选，分页返回。
func (s *RecordService) List(ctx context.Context, query RecordListQuery) (*common.ListResult, error) {
	var rows []TCaliRecord
	var total int64

	q := s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("record_no like ?", "%"+query.RecordNo+"%")

	q.Count(&total)
	q.Limit(10).Offset(0).Where("status = ?", 0).Find(&rows)

	return &common.ListResult{List: rows, Total: total}, nil
}

type RecordListQuery struct {
	RecordNo string
	Status   *int
	Page     common.PageQuery
}
