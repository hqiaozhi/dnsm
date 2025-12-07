package dns

import (
	logic "dnsm/internal/logic/dns"
	"dnsm/internal/svc"

	"github.com/gin-gonic/gin"
)

type IDNS interface {
	// QueryDomain 列出所有域名
	QueryDomain(c *gin.Context)
	// QueryDomainWithPagination 分页查询域名列表，包含记录数量
	QueryDomainWithPagination(c *gin.Context)
	// GetDomain 获取单个域名详情
	GetDomain(c *gin.Context)
	// CreateDomain 创建/更新域名
	CreateDomain(c *gin.Context)
	// DeleteDomain 删除域名
	DeleteDomain(c *gin.Context)
	// GetRecords 获取域名下所有记录
	GetRecords(c *gin.Context)
	// AddRecord 添加解析记录
	AddRecord(c *gin.Context)
	// UpdateRecord 更新解析记录
	UpdateRecord(c *gin.Context)
	// DeleteRecord 删除解析记录
	DeleteRecord(c *gin.Context)
}

type DNS struct {
	svcCtx *svc.SvcContext
	dns    *logic.DNSLogic
}

func New(svcCtx *svc.SvcContext) IDNS {
	return &DNS{
		svcCtx: svcCtx,
		dns:    logic.New(svcCtx),
	}
}
