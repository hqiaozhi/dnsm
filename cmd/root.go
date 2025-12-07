package cmd

import (
	"dnsm/internal/router"
	"dnsm/internal/svc"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var RootCmd = &cobra.Command{
	Short: "app [flag] [args]",
	Run:   startRun,
}

func startRun(cmd *cobra.Command, args []string) {
	svcCtx := svc.NewSvcContext()
	// 启动DNS服务
	go func() {
		time.Sleep(3 * time.Second)
		log.Println("Starting DNS service...")
		err := svcCtx.DNSEngine.Start()
		if err != nil {
			log.Fatalf("Failed to start DNS server: %v", err)
		}
	}()

	go func() {
		// 等待停止信号
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan
		err := svcCtx.DNSEngine.Stop()
		if err != nil {
			log.Printf("Failed to stop DNS server: %v", err)
		}
		log.Println("DNS server stopped")
	}()

	appcfg := svcCtx.Conf.Gin
	addr := appcfg.Host + ":" + strconv.Itoa(appcfg.Port)
	engine := router.New(addr, appcfg.Mode, svcCtx)

	// 2. 注册业务路由（核心：解耦路由定义与引擎实现）
	engine.RegisterRoutes(router.RegisterBusinessRoutes)

	// 3. 启动服务器（阻塞，支持优雅关闭）
	engine.Run()
}

func init() {
	// 初始化根命令，这一步会自动添加 completion 命令
	RootCmd.CompletionOptions.DisableDefaultCmd = true
}
