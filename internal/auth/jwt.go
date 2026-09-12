package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IssueToken 用 HMAC-SHA256 签发登录令牌。
func IssueToken(secret string, expireHours int, id Identity) (string, error) {
	if expireHours <= 0 {
		expireHours = 12
	}
	claims := jwt.MapClaims{
		"sub":      id.UserID,
		"username": id.UserName,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Duration(expireHours) * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

// ParseToken 校验并取出登录身份。
func ParseToken(secret, tokenStr string) (Identity, error) {
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("签名算法不被接受")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return Identity{}, errors.New("令牌无效")
	}
	id := Identity{}
	switch v := claims["sub"].(type) {
	case float64:
		id.UserID = uint64(v)
	}
	if name, ok := claims["username"].(string); ok {
		id.UserName = name
	}
	if id.UserName == "" {
		return Identity{}, errors.New("令牌缺少登录人")
	}
	return id, nil
}
