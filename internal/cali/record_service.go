package cali

import (
	"context"
	"errors"
	"math"

	"gorm.io/gorm"

	"metrobase/internal/common"
	"metrobase/internal/db"
)

// RecordService 校准记录登记。
type RecordService struct {
	db *gorm.DB
}

func NewRecordService(db *gorm.DB) *RecordService { return &RecordService{db: db} }

type RecordInput struct {
	RecordNo       string  `json:"recordNo"`
	DeviceID       uint64  `json:"deviceId"`
	CalibDate      string  `json:"calibDate"`
	StandardValue  float64 `json:"standardValue"`
	IndicatedValue float64 `json:"indicatedValue"`
	Remark         string  `json:"remark"`
}

// Create 登记一条校准记录并给出合格结论。
func (s *RecordService) Create(ctx context.Context, in RecordInput) (*TCaliRecord, error) {
	if in.RecordNo == "" || in.DeviceID == 0 || in.CalibDate == "" {
		return nil, common.NewBizError("记录编号、器具、校准日期不能为空")
	}
	dev, err := s.mustDevice(ctx, in.DeviceID)
	if err != nil {
		return nil, err
	}
	var dup int64
	s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("record_no = ?", in.RecordNo).Count(&dup)
	if dup > 0 {
		return nil, common.NewBizError("记录编号已存在")
	}

	refErr, result, err := judge(dev, in.StandardValue, in.IndicatedValue)
	if err != nil {
		return nil, err
	}

	rec := TCaliRecord{
		RecordNo:       in.RecordNo,
		DeviceID:       dev.ID,
		DeviceNo:       dev.DeviceNo,
		DeviceName:     dev.DeviceName,
		CalibDate:      in.CalibDate,
		StandardValue:  in.StandardValue,
		IndicatedValue: in.IndicatedValue,
		ReferenceError: refErr,
		Result:         result,
		Status:         0,
		Remark:         in.Remark,
	}
	if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// Update 编辑校准记录：器具、标准值、示值变化后重算误差与结论。
func (s *RecordService) Update(ctx context.Context, id uint64, in RecordInput) error {
	if in.RecordNo == "" || in.DeviceID == 0 || in.CalibDate == "" {
		return common.NewBizError("记录编号、器具、校准日期不能为空")
	}
	dev, err := s.mustDevice(ctx, in.DeviceID)
	if err != nil {
		return err
	}
	refErr, result, err := judge(dev, in.StandardValue, in.IndicatedValue)
	if err != nil {
		return err
	}
	tx := s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"record_no":       in.RecordNo,
			"device_id":       dev.ID,
			"device_no":       dev.DeviceNo,
			"device_name":     dev.DeviceName,
			"calib_date":      in.CalibDate,
			"standard_value":  in.StandardValue,
			"indicated_value": in.IndicatedValue,
			"reference_error": refErr,
			"result":          result,
			"remark":          in.Remark,
		})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return common.NewBizError("校准记录不存在")
	}
	return nil
}

// Issue 发证：回填证书编号，状态由「已登记」转为「已发证」。
func (s *RecordService) Issue(ctx context.Context, id uint64, certNo string) error {
	if certNo == "" {
		return common.NewBizError("证书编号不能为空")
	}
	tx := s.db.WithContext(ctx).Model(&TCaliRecord{}).
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{"cert_no": certNo, "status": 1})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return common.NewBizError("记录不存在或已发证")
	}
	return nil
}

// List 列表查询：记录编号模糊 + 状态筛选，分页返回。
func (s *RecordService) List(ctx context.Context, query RecordListQuery) (*common.ListResult, error) {
	var rows []TCaliRecord
	var total int64

	q := s.db.WithContext(ctx).Clauses(db.Read()).Model(&TCaliRecord{})
	if query.RecordNo != "" {
		q = q.Where("record_no like ?", "%"+query.RecordNo+"%")
	}
	if query.Status != nil {
		q = q.Where("status = ?", *query.Status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := q.Scopes(common.Paginate(query.Page)).
		Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return &common.ListResult{List: rows, Total: total}, nil
}

// mustDevice 取器具档案，不存在时返回业务错误。
func (s *RecordService) mustDevice(ctx context.Context, id uint64) (*TCaliDevice, error) {
	var dev TCaliDevice
	err := s.db.WithContext(ctx).First(&dev, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, common.NewBizError("器具档案不存在")
	}
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

// judge 计算引用误差（|示值-标准值|/量程上限×100%，保留两位小数），
// 对照档案准确度等级判定：不超过为合格，超出为不合格。
func judge(dev *TCaliDevice, standard, indicated float64) (float64, int, error) {
	if dev.FullScale <= 0 {
		return 0, 0, common.NewBizError("器具档案未维护量程上限，无法判定")
	}
	refErr := math.Round(math.Abs(indicated-standard)/dev.FullScale*100*100) / 100
	if refErr <= dev.AccuracyClass {
		return refErr, 0, nil
	}
	return refErr, 1, nil
}

type RecordListQuery struct {
	RecordNo string
	Status   *int
	Page     common.PageQuery
}
