package router

import (
	"dnsm/internal/handler/dns"
	"dnsm/internal/handler/user"
	"dnsm/internal/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterBusinessRoutes(engine *GinEngine) {
	engine.Use(gin.Logger(), middleware.Cors())
	ctx := engine.svcCtx

	// 健康检查路由
	engine.ginEngine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// 版本：/api/v1
	v1 := engine.Group("/api/v1")
	{
		// 公共路由组
		publicGroup := v1.Group("")
		{
			publicGroup.POST("/user/login", user.New(ctx).Login)
		}

		// 需权限校验
		authGroup := v1.Group("/dns")
		authGroup.Use(middleware.Auth(ctx))
		{
			publicGroup.POST("/user/logout", user.New(ctx).Logout)
		}

		{
			// 域名相关接口
			authGroup.GET("", dns.New(ctx).QueryDomain)                    // 列出所有域名
			authGroup.GET("/page", dns.New(ctx).QueryDomainWithPagination) // 分页查询域名列表，包含记录数量
			authGroup.GET("/:domain", dns.New(ctx).GetDomain)              // 获取单个域名详情
			authGroup.POST("", dns.New(ctx).CreateDomain)                  // 创建/更新域名
			authGroup.DELETE("/:domain", dns.New(ctx).DeleteDomain)        // 删除域名

			// 记录相关接口
			authGroup.GET("/:domain/records", dns.New(ctx).GetRecords)              // 获取域名下所有记录
			authGroup.POST("/:domain/records", dns.New(ctx).AddRecord)              // 添加解析记录
			authGroup.PUT("/:domain/records/:record", dns.New(ctx).UpdateRecord)    // 更新解析记录
			authGroup.DELETE("/:domain/records/:record", dns.New(ctx).DeleteRecord) // 删除解析记录
		}
	}
}
