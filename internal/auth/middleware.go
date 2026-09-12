package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"metrobase/internal/response"
)

// Middleware 校验 Authorization: Bearer <token>，并把登录身份放进请求 context。
// 白名单路径（/api/login 等）由调用方在注册路由时绕过本中间件。
func Middleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenStr := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		id, err := ParseToken(secret, tokenStr)
		if err != nil {
			response.FailWithCode(c, 401, "未登录或登录已失效")
			c.Abort()
			return
		}
		ctx := WithIdentity(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Set("loginName", id.UserName)
		c.Next()
	}
}
