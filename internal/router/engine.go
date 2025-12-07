package router

import (
	"context"
	"dnsm/internal/svc"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var assets embed.FS

func Assets() (http.FileSystem, http.FileSystem) {
	consoleFS, err := fs.Sub(assets, "dist")
	if err != nil {
		panic("获取 console 资源失败: " + err.Error())
	}
	assetsFS, err := fs.Sub(assets, "dist/assets")
	if err != nil {
		panic(fmt.Sprintf("初始化 assets FS 失败: %v", err))
	}
	return http.FS(consoleFS), http.FS(assetsFS)
}

// Engine 路由引擎结构体
type GinEngine struct {
	ginEngine *gin.Engine  // 底层 Gin 引擎
	server    *http.Server // HTTP 服务器（用于优雅关闭）
	addr      string       // 监听地址（如 :8080）
	svcCtx    *svc.SvcContext
}

// NewEngine 创建路由引擎实例
// addr: 监听地址（如 ":8080"）
// mode: Gin 运行模式（gin.DebugMode/gin.ReleaseMode/gin.TestMode）
func New(addr, mode string, svcCtx *svc.SvcContext) *GinEngine {
	gin.SetMode(mode)
	engin := gin.New()
	engin.Use(gin.Recovery())

	// 注册静态文件服务 - 使用Gin的StaticFS方法
	consoleFS, assetsFS := Assets()
	engin.StaticFS("/assets", assetsFS)
	engin.NoRoute(func(c *gin.Context) {
		// 从 embed 中读取 index.html（适配你的打包方式）
		content, err := consoleFS.Open("index.html")
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		defer content.Close()
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, content); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, buf.String())
	})

	return &GinEngine{
		ginEngine: engin, // 默认包含 Logger 和 Recovery 中间件
		addr:      addr,
		svcCtx:    svcCtx,
	}
}

// Use 注册全局中间件（对所有路由生效）
func (e *GinEngine) Use(middlewares ...gin.HandlerFunc) {
	e.ginEngine.Use(middlewares...)
}

// Group 创建路由组（支持路由前缀和组内中间件）
func (e *GinEngine) Group(relativePath string, middlewares ...gin.HandlerFunc) *gin.RouterGroup {
	return e.ginEngine.Group(relativePath, middlewares...)
}

// RegisterRoutes 注册业务路由（由外部实现，解耦核心逻辑）
func (e *GinEngine) RegisterRoutes(registerFunc func(engine *GinEngine)) {
	registerFunc(e)
}

// Run 启动服务器（阻塞）并支持优雅关闭
func (e *GinEngine) Run() {
	// 初始化 HTTP 服务器
	e.server = &http.Server{
		Addr:    e.addr,
		Handler: e.ginEngine,
	}

	// 启动服务器（非阻塞）
	go func() {
		log.Printf("The gin server has started successfully. Listen for the address: %s (Mode: %s)\n", e.addr, gin.Mode())
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("The gin server startup failed: %v", err)
		}
	}()

	// 监听关闭信号（SIGINT: Ctrl+C，SIGTERM: 容器停止信号）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞等待信号
	log.Println("The gin server is shutting down gracefully...")

	// 创建 5 秒超时上下文（确保请求有足够时间处理）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器（不再接收新请求，等待现有请求完成）
	if err := e.server.Shutdown(ctx); err != nil {
		log.Printf("The gin server shutdown failed: %v", err)
	} else {
		log.Println("The gin server has been shut down gracefully.")
	}
}
