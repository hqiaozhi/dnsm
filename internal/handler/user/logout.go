package user

import "github.com/gin-gonic/gin"

func (u *User) Logout(c *gin.Context) {
	u.svcCtx.RESP.RESP_OK(c)
}
