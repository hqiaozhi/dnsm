package dns

import (
	"context"
	"dnsm/internal/core"
)

// QueryDomain 列出所有域名
func (d *DNSLogic) QueryDomain(ctx context.Context) []string {
	domains := d.svcCtx.DNSManager.ListDomains()
	return domains
}

// QueryDomainWithPagination 分页查询域名列表，包含记录数量
func (d *DNSLogic) QueryDomainWithPagination(ctx context.Context, page, pageSize int) (core.DomainListResult, error) {
	return d.svcCtx.DNSManager.ListDomainsWithPagination(page, pageSize)
}

// GetDomain 获取单个域名信息
func (d *DNSLogic) GetDomain(ctx context.Context, domainName string) (core.Domain, error) {
	return d.svcCtx.DNSManager.GetDomain(domainName)
}

// CreateDomain 创建/更新域名
func (d *DNSLogic) CreateDomain(ctx context.Context, domain core.Domain) error {
	return d.svcCtx.DNSManager.AddOrUpdateDomain(domain)
}

// DeleteDomain 删除域名
func (d *DNSLogic) DeleteDomain(ctx context.Context, domainName string) error {
	return d.svcCtx.DNSManager.DeleteDomain(domainName)
}

// GetRecords 获取域名下所有记录
func (d *DNSLogic) GetRecords(ctx context.Context, domainName string) ([]core.Record, error) {
	return d.svcCtx.DNSManager.GetRecords(domainName)
}

// AddRecord 添加解析记录
func (d *DNSLogic) AddRecord(ctx context.Context, domainName string, record core.Record) error {
	return d.svcCtx.DNSManager.AddRecord(domainName, record)
}

// UpdateRecord 更新解析记录
func (d *DNSLogic) UpdateRecord(ctx context.Context, domainName, recordName string, record core.Record) error {
	return d.svcCtx.DNSManager.UpdateRecord(domainName, recordName, record)
}

// DeleteRecord 删除解析记录
func (d *DNSLogic) DeleteRecord(ctx context.Context, domainName, recordName string) error {
	return d.svcCtx.DNSManager.DeleteRecord(domainName, recordName)
}
