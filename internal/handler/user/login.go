package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (u *User) Login(c *gin.Context) {
	// 判断请求内容是否为空
	if c.Request.ContentLength == 0 {
		u.svcCtx.RESP.RESP_ERROR(c, http.StatusBadRequest, "请求内容不能为空")
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		u.svcCtx.RESP.RESP_ERROR(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	// 校验用户名密码（直接从配置文件获取进行简单校验）
	userconf := u.svcCtx.Conf.Login
	if req.Username != userconf.Username || req.Password != userconf.Password {
		u.svcCtx.RESP.RESP_ERROR(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := u.user.Login(req.Username)
	if err != nil {
		u.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, "token生成失败")
		return
	}

	data := map[string]string{
		"token": token,
	}
	u.svcCtx.RESP.RESP_DATA(c, data)
}
