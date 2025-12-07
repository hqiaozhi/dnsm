package svc

import (
	"dnsm/internal/conf"
	"dnsm/internal/core"
	"dnsm/internal/utils/jwt"
	"dnsm/internal/utils/resp"
	"log"
)

type SvcContext struct {
	Conf       *conf.Config
	DNSEngine  *core.DNSEngine
	DNSManager core.DNSManager
	RESP       *resp.Resp
	JWT        *jwt.JwtService
}

func NewSvcContext() *SvcContext {
	s := &SvcContext{}

	// 加载配置
	config, v, configPath := conf.New()
	config.WatchConfigChanges(v)
	s.Conf = config

	// 初始化DNS管理器
	s.DNSManager = core.NewViperYAMLManager(v, configPath)
	if err := s.DNSManager.Load(); err != nil {
		log.Fatalf("Failed to load DNS configuration: %v", err)
	}

	// 初始化DNS引擎
	s.DNSEngine = core.New(config)

	// 响应
	s.RESP = resp.New()

	// JWT
	s.JWT = jwt.NewJWTService(&config.JWT)

	return s
}
