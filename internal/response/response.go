package response

import (
	"github.com/gin-gonic/gin"

	"metrobase/internal/common"
)

// R 统一返回结构：code/msg/data。
type R struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

const (
	CodeOK   = 0
	CodeFail = 500
)

func OK(c *gin.Context, data any) {
	c.JSON(200, R{Code: CodeOK, Msg: "成功", Data: data})
}

func Fail(c *gin.Context, msg string) {
	FailWithCode(c, CodeFail, msg)
}

func FailWithCode(c *gin.Context, code int, msg string) {
	c.JSON(200, R{Code: code, Msg: msg})
}

// Recovery 兜底异常：任何 panic 也必须返回统一结构，不向前端泄漏堆栈。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				Fail(c, "服务繁忙，请稍后再试")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// AbortBiz Service 返回 BizError 时走统一业务失败结构；其余错误按系统错误处理。
func AbortBiz(c *gin.Context, err error) {
	if biz, ok := err.(*common.BizError); ok {
		Fail(c, biz.Msg)
		return
	}
	Fail(c, "服务繁忙，请稍后再试")
}
