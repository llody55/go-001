package sys

import (
	"github.com/gin-gonic/gin"

	"metrobase/internal/response"
)

type Controller struct{ auth *AuthService }

func NewController(auth *AuthService) *Controller { return &Controller{auth: auth} }

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 白名单接口：POST /api/login。
func (ctl *Controller) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "用户名和密码不能为空")
		return
	}
	token, err := ctl.auth.Login(req.Username, req.Password)
	if err != nil {
		response.AbortBiz(c, err)
		return
	}
	response.OK(c, gin.H{"token": token})
}
