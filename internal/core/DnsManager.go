package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// -------------------------- 基础数据结构 --------------------------
// Record DNS解析记录结构体（与配置文件映射）
type Record struct {
	Name  string `mapstructure:"name" yaml:"name"` // 记录名（子域名/反向IP）
	Type  string `mapstructure:"type" yaml:"type"` // 解析类型 A/AAAA/PTR/CNAME 等
	Value string `mapstructure:"value" yaml:"value"`
	TTL   int    `mapstructure:"ttl" yaml:"ttl"`
}

// Domain 域名结构体（包含归属的解析记录）
type Domain struct {
	Name    string   `mapstructure:"name" yaml:"name"`
	Records []Record `mapstructure:"records" yaml:"records"`
}

// DomainInfo 域名信息结构体（用于列表展示，包含记录数量）
type DomainInfo struct {
	Name        string `json:"name"`         // 域名名称
	RecordCount int    `json:"record_count"` // 记录数量
}

// DomainListResult 域名列表查询结果
type DomainListResult struct {
	Total   int64        `json:"total"`   // 总域名数
	Domains []DomainInfo `json:"domains"` // 当前页域名列表
}

// -------------------------- 核心接口定义 --------------------------
// DNSManager DNS管理器核心接口（抽象所有操作）
type DNSManager interface {
	// 加载配置（从Viper/配置文件初始化数据）
	Load() error

	// 域名级操作
	AddOrUpdateDomain(domain Domain) error       // 新增/更新域名
	DeleteDomain(domainName string) error        // 删除域名
	GetDomain(domainName string) (Domain, error) // 查询单个域名完整信息

	// 解析记录级操作
	AddRecord(domainName string, record Record) error                // 新增解析记录
	UpdateRecord(domainName, recordName string, record Record) error // 更新解析记录
	DeleteRecord(domainName, recordName string) error                // 删除解析记录
	GetRecords(domainName string) ([]Record, error)                  // 查询域名下所有记录

	// 辅助操作
	ListDomains() []string                                                  // 列出所有已加载的域名
	ListDomainsWithPagination(page, pageSize int) (DomainListResult, error) // 分页查询域名列表，包含记录数量
}

// -------------------------- 接口实现：ViperYAMLManager --------------------------
// ViperYAMLManager DNS管理器的Viper+YAML实现（无损修改配置）
type ViperYAMLManager struct {
	mu           sync.RWMutex      // 并发安全锁
	domainMap    map[string]Domain // 内存映射：域名->解析记录
	viper        *viper.Viper      // Viper配置实例
	configPath   string            // 配置文件路径
	fullYAMLNode *yaml.Node        // 完整YAML节点树（保留所有配置）
}

// NewViperYAMLManager 创建ViperYAMLManager实例（接口工厂方法）
func NewViperYAMLManager(v *viper.Viper, configPath string) DNSManager {
	return &ViperYAMLManager{
		domainMap:  make(map[string]Domain),
		viper:      v,
		configPath: configPath,
	}
}

// -------------------------- 实现DNSManager接口 --------------------------
// Load 加载配置（实现接口）
func (m *ViperYAMLManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 读取完整YAML文件，保留所有节点
	yamlData, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	var rootNode yaml.Node
	if err := yaml.Unmarshal(yamlData, &rootNode); err != nil {
		return fmt.Errorf("解析YAML节点失败: %w", err)
	}
	m.fullYAMLNode = &rootNode

	// 2. 从Viper解析domains到内存映射
	var domains []Domain
	if err := m.viper.UnmarshalKey("domains", &domains); err != nil {
		return fmt.Errorf("解析domains节点失败: %w", err)
	}

	// 3. 构建内存映射
	m.domainMap = make(map[string]Domain, len(domains))
	for _, domain := range domains {
		m.domainMap[domain.Name] = domain
	}
	return nil
}

// AddOrUpdateDomain 新增/更新域名（实现接口）
func (m *ViperYAMLManager) AddOrUpdateDomain(domain Domain) error {
	if domain.Name == "" {
		return fmt.Errorf("域名名称不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新内存映射
	m.domainMap[domain.Name] = domain

	// 无损更新YAML配置
	return m.updateDomainsNode()
}

// DeleteDomain 删除域名（实现接口）
func (m *ViperYAMLManager) DeleteDomain(domainName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.domainMap[domainName]; !exists {
		return fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 删除内存映射
	delete(m.domainMap, domainName)

	// 无损更新YAML配置
	return m.updateDomainsNode()
}

// GetDomain 查询单个域名完整信息（实现接口）
func (m *ViperYAMLManager) GetDomain(domainName string) (Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domain, exists := m.domainMap[domainName]
	if !exists {
		return Domain{}, fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 返回副本，避免外部修改内部数据
	records := make([]Record, len(domain.Records))
	copy(records, domain.Records)
	domain.Records = records
	return domain, nil
}

// AddRecord 新增解析记录（实现接口）
func (m *ViperYAMLManager) AddRecord(domainName string, record Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查域名是否存在
	domain, exists := m.domainMap[domainName]
	if !exists {
		return fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 检查记录是否重复（同名同类型）
	for _, r := range domain.Records {
		if r.Name == record.Name && r.Type == record.Type {
			return fmt.Errorf("域名 %s 下已存在记录 %s(%s)", domainName, record.Name, record.Type)
		}
	}

	// 新增记录
	domain.Records = append(domain.Records, record)
	m.domainMap[domainName] = domain

	// 无损更新YAML配置
	return m.updateDomainsNode()
}

// UpdateRecord 更新解析记录（实现接口）
func (m *ViperYAMLManager) UpdateRecord(domainName, recordName string, newRecord Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查域名是否存在
	domain, exists := m.domainMap[domainName]
	if !exists {
		return fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 查找并更新记录
	found := false
	for i, r := range domain.Records {
		if r.Name == recordName {
			domain.Records[i] = newRecord
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("域名 %s 下不存在记录 %s", domainName, recordName)
	}

	// 更新内存映射
	m.domainMap[domainName] = domain

	// 无损更新YAML配置
	return m.updateDomainsNode()
}

// DeleteRecord 删除解析记录（实现接口）
func (m *ViperYAMLManager) DeleteRecord(domainName, recordName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查域名是否存在
	domain, exists := m.domainMap[domainName]
	if !exists {
		return fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 过滤要删除的记录
	newRecords := make([]Record, 0, len(domain.Records))
	found := false
	for _, r := range domain.Records {
		if r.Name != recordName {
			newRecords = append(newRecords, r)
		} else {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("域名 %s 下不存在记录 %s", domainName, recordName)
	}

	// 更新内存映射
	domain.Records = newRecords
	m.domainMap[domainName] = domain

	// 无损更新YAML配置
	return m.updateDomainsNode()
}

// GetRecords 查询域名下所有记录（实现接口）
func (m *ViperYAMLManager) GetRecords(domainName string) ([]Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domain, exists := m.domainMap[domainName]
	if !exists {
		return nil, fmt.Errorf("域名 %s 不存在", domainName)
	}

	// 返回副本，避免外部修改
	records := make([]Record, len(domain.Records))
	copy(records, domain.Records)
	return records, nil
}

// ListDomains 列出所有域名（实现接口）
func (m *ViperYAMLManager) ListDomains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domains := make([]string, 0, len(m.domainMap))
	for name := range m.domainMap {
		domains = append(domains, name)
	}
	return domains
}

// ListDomainsWithPagination 分页查询域名列表，包含记录数量（实现接口）
func (m *ViperYAMLManager) ListDomainsWithPagination(page, pageSize int) (DomainListResult, error) {
	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // 默认每页20条
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取总域名数
	total := int64(len(m.domainMap))

	// 创建域名信息列表
	domainInfos := make([]DomainInfo, 0, len(m.domainMap))
	for name, domain := range m.domainMap {
		domainInfos = append(domainInfos, DomainInfo{
			Name:        name,
			RecordCount: len(domain.Records),
		})
	}

	// 按域名名称字母排序
	sort.Slice(domainInfos, func(i, j int) bool {
		return strings.ToLower(domainInfos[i].Name) < strings.ToLower(domainInfos[j].Name)
	})

	// 计算分页范围
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(domainInfos) {
		// 超出范围，返回空列表
		return DomainListResult{
			Total:   total,
			Domains: []DomainInfo{},
		}, nil
	}
	if end > len(domainInfos) {
		end = len(domainInfos)
	}

	// 返回分页结果
	return DomainListResult{
		Total:   total,
		Domains: domainInfos[start:end],
	}, nil
}

// -------------------------- 私有辅助方法 --------------------------
// updateDomainsNode 更新YAML中的domains节点（使用viper直接更新配置）
func (m *ViperYAMLManager) updateDomainsNode() error {
	// 1. 将内存映射转换为[]Domain
	domains := make([]Domain, 0, len(m.domainMap))
	for _, domain := range m.domainMap {
		domains = append(domains, domain)
	}

	// 2. 使用viper直接设置domains配置
	m.viper.Set("domains", domains)

	// 3. 写回配置文件
	if err := m.viper.WriteConfig(); err != nil {
		// 如果WriteConfig失败（可能是因为文件权限问题），尝试使用SafeWriteConfig
		if err := m.viper.SafeWriteConfig(); err != nil {
			// 如果都失败，尝试直接写入文件
			allConfig := make(map[string]interface{})
			for _, key := range m.viper.AllKeys() {
				allConfig[key] = m.viper.Get(key)
			}

			// 确保配置目录存在
			configDir := filepath.Dir(m.configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("创建配置目录失败: %w", err)
			}

			// 直接将所有配置写入YAML文件
			file, err := os.OpenFile(m.configPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("打开配置文件失败: %w", err)
			}
			defer file.Close()

			yamlEncoder := yaml.NewEncoder(file)
			yamlEncoder.SetIndent(2)
			if err := yamlEncoder.Encode(allConfig); err != nil {
				yamlEncoder.Close()
				return fmt.Errorf("序列化YAML失败: %w", err)
			}
			yamlEncoder.Close()
		}
	}

	// 4. 重新加载Viper保证数据最新
	return m.viper.ReadInConfig()
}
