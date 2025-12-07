package user

import (
	logic "dnsm/internal/logic/user"
	"dnsm/internal/svc"

	"github.com/gin-gonic/gin"
)

type Iuser interface {
	Login(c *gin.Context)
	Logout(c *gin.Context)
}

type User struct {
	svcCtx *svc.SvcContext
	user   *logic.User
}

func New(svcCtx *svc.SvcContext) Iuser {
	return &User{
		svcCtx: svcCtx,
		user:   logic.New(svcCtx),
	}
}
