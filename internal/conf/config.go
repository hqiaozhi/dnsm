package conf

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const configFileName = "config"

type DNSConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type Record struct {
	Name  string `mapstructure:"name"`
	Type  string `mapstructure:"type"`
	Value string `mapstructure:"value"`
	TTL   int    `mapstructure:"ttl"`
}

type Domain struct {
	Name    string   `mapstructure:"name"`
	Records []Record `mapstructure:"records"`
}

type JWTConfig struct {
	SecretKey     string        `mapstructure:"secret_key"`     // 密钥（必须保密）
	Issuer        string        `mapstructure:"issuer"`         // 签发者
	Audience      string        `mapstructure:"audience"`       // 受众
	ExpireHours   time.Duration `mapstructure:"expire_hours"`   // 过期时间（小时）
	RefreshHours  time.Duration `mapstructure:"refresh_hours"`  // 刷新令牌过期时间（小时）
	SigningMethod string        `mapstructure:"signing_method"` // 签名算法（HS256/HS512）
}

type LoginUser struct {
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
}

type GinConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	Mode               string        `mapstructure:"mode"`                 // 运行模式（debug/release/test）
	ReadTimeout        time.Duration `mapstructure:"read_timeout"`         // 读取超时
	WriteTimeout       time.Duration `mapstructure:"write_timeout"`        // 写入超时
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`         // 空闲超时
	MaxMultipartMemory int64         `mapstructure:"max_multipart_memory"` // 最大上传内存
}

type Config struct {
	Server   DNSConfig `mapstructure:"server"`
	Upstream []string  `mapstructure:"upstream"`
	Domains  []Domain  `mapstructure:"domains"`
	JWT      JWTConfig `mapstructure:"jwt"`
	Gin      GinConfig `mapstructure:"gin"`
	Login    LoginUser `mapstructure:"login"`
}

// GetUpstream 获取上游DNS服务器列表（暂时简化）
func (c *Config) GetUpstream() []string {
	// 返回副本，避免外部修改
	upstreamCopy := make([]string, len(c.Upstream))
	copy(upstreamCopy, c.Upstream)
	return upstreamCopy
}

// GetDomains 获取域名配置列表（暂时简化）
func (c *Config) GetDomains() []Domain {
	// 返回副本，避免外部修改
	domainsCopy := make([]Domain, len(c.Domains))
	for i, domain := range c.Domains {
		domainsCopy[i] = Domain{
			Name:    domain.Name,
			Records: append([]Record{}, domain.Records...),
		}
	}
	return domainsCopy
}

// GetServer 获取服务器配置（暂时简化）
func (c *Config) GetServer() DNSConfig {
	return c.Server
}

// LoadConfig 初始化并加载配置
func New() (*Config, *viper.Viper, string) {
	// 设置 viper 读取 config.yaml
	v := viper.New()
	v.SetConfigName(configFileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/dnsm/")
	v.AddConfigPath("./conf")

	// 创建配置变量
	var config Config

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在，使用默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using default configuration")

			// 使用默认配置
			config = Config{
				Server: DNSConfig{
					Host: "0.0.0.0",
					Port: 53,
				},
				Upstream: []string{
					"223.5.5.5:53",
					"223.6.6.6:53",
				},
				Domains: []Domain{},
			}
			// 当配置文件不存在时，使用默认的配置文件路径
			configPath := "./config.yaml"
			return &config, v, configPath
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		// 配置文件存在，解析它
		if err := v.Unmarshal(&config); err != nil {
			log.Fatalf("Unable to decode into struct: %v", err)
		}
	}
	configPath := v.ConfigFileUsed()
	log.Println("Initial configuration loaded successfully.")

	return &config, v, configPath
}

// WatchConfigChanges 启动一个 goroutine 来监听配置文件变化并自动重新加载
func (c *Config) WatchConfigChanges(v *viper.Viper) {
	// 注意：此方法现在使用的是全局viper实例，在实际使用中应该传入正确的viper实例
	// 为了兼容性暂时保留此实现
	if v.ConfigFileUsed() != "" {
		v.WatchConfig()

		// 设置配置变化时的回调函数
		v.OnConfigChange(func(e fsnotify.Event) {
			log.Printf("Config file changed: %s Op: %s", e.Name, e.Op.String())

			// 尝试重新加载配置到临时变量
			if err := v.Unmarshal(&c); err != nil {
				log.Printf("Error unmarshalling updated config: %v. Keeping old config.", err)
				return // 如果新配置有错误，保持旧配置不变
			}
			log.Println("Configuration reloaded successfully and applied.")
		})
	} else {
		log.Println("No config file to watch, skipping config watching.")
	}
}
