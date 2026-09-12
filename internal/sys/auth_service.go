package sys

import (
	"crypto/md5"
	"encoding/hex"

	"gorm.io/gorm"

	"metrobase/internal/auth"
	"metrobase/internal/common"
	"metrobase/internal/db"
)

type AuthService struct {
	db     *gorm.DB
	secret string
	hours  int
}

func NewAuthService(db *gorm.DB, secret string, expireHours int) *AuthService {
	return &AuthService{db: db, secret: secret, hours: expireHours}
}

// MD5 单次迭代后入库比对。
func MD5(raw string) string {
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Login 校验账号口令并签发 token。
func (s *AuthService) Login(username, password string) (string, error) {
	var user TSysUser
	err := s.db.Clauses(db.Read()).
		Where("username = ?", username).First(&user).Error
	if err != nil {
		return "", common.NewBizError("用户名或密码不正确")
	}
	if user.Status == 1 {
		return "", common.NewBizError("账号已停用")
	}
	if user.Password != MD5(password) {
		return "", common.NewBizError("用户名或密码不正确")
	}
	return auth.IssueToken(s.secret, s.hours, auth.Identity{UserID: user.ID, UserName: user.Username})
}
