package auth

import "context"

// Identity 登录身份。审计列只以 context 中的登录身份为准，
// 接口入参里的“操作人/用户名”不作为身份来源。
type Identity struct {
	UserID   uint64
	UserName string
}

type ctxKey struct{}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// CurrentIdentity 未登录时返回零值。
func CurrentIdentity(ctx context.Context) Identity {
	if v, ok := ctx.Value(ctxKey{}).(Identity); ok {
		return v
	}
	return Identity{}
}

func CurrentUserName(ctx context.Context) string {
	return CurrentIdentity(ctx).UserName
}
