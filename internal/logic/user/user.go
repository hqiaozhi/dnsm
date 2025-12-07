package user

import (
	"dnsm/internal/svc"
	"dnsm/internal/utils/jwt"
)

type User struct {
	svcCtx *svc.SvcContext
	jwt    *jwt.JwtService
}

func New(svcCtx *svc.SvcContext) *User {
	return &User{
		svcCtx: svcCtx,
		jwt:    svcCtx.JWT,
	}
}
