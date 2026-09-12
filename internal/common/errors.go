package common

// BizError 业务异常：经统一返回中间件输出为标准错误结构，不向调用方泄漏堆栈。
type BizError struct {
	Msg string
}

func (e *BizError) Error() string { return e.Msg }

func NewBizError(msg string) *BizError { return &BizError{Msg: msg} }
