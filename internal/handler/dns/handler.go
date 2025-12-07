package dns

import (
	"dnsm/internal/core"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Query 列出所有域名
func (d *DNS) QueryDomain(c *gin.Context) {
	domains := d.dns.QueryDomain(c)

	var data struct {
		Items []string `json:"items"`
		Total int      `json:"total"`
	}
	data.Items = domains
	data.Total = len(data.Items)
	d.svcCtx.RESP.RESP_DATA(c, data)
}

// QueryDomainWithPagination 分页查询域名列表，包含记录数量
func (d *DNS) QueryDomainWithPagination(c *gin.Context) {
	// 获取分页参数，设置默认值
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	// 转换为整数
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 调用逻辑层方法
	result, err := d.dns.QueryDomainWithPagination(c, page, pageSize)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	d.svcCtx.RESP.RESP_DATA(c, result)
}

// GetDomain 获取单个域名详情
func (d *DNS) GetDomain(c *gin.Context) {
	domainName := c.Param("domain")
	if domainName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名参数不能为空")
		return
	}

	domain, err := d.dns.GetDomain(c, domainName)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusNotFound, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_DATA(c, domain)
}

// CreateDomain 创建/更新域名
func (d *DNS) CreateDomain(c *gin.Context) {
	var req core.Domain
	if err := c.ShouldBindJSON(&req); err != nil {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "请求参数格式错误: "+err.Error())
		return
	}

	if req.Name == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名名称不能为空")
		return
	}

	err := d.dns.CreateDomain(c, req)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_OK(c)
}

// DeleteDomain 删除域名
func (d *DNS) DeleteDomain(c *gin.Context) {
	domainName := c.Param("domain")
	if domainName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名参数不能为空")
		return
	}

	err := d.dns.DeleteDomain(c, domainName)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_OK(c)
}

// GetRecords 获取域名下所有记录
func (d *DNS) GetRecords(c *gin.Context) {
	domainName := c.Param("domain")
	if domainName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名参数不能为空")
		return
	}

	records, err := d.dns.GetRecords(c, domainName)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusNotFound, err.Error())
		return
	}

	var data struct {
		Items []core.Record `json:"items"`
		Total int           `json:"total"`
	}
	data.Items = records
	data.Total = len(data.Items)
	d.svcCtx.RESP.RESP_DATA(c, data)
}

// AddRecord 添加解析记录
func (d *DNS) AddRecord(c *gin.Context) {
	domainName := c.Param("domain")
	if domainName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名参数不能为空")
		return
	}

	var req core.Record
	if err := c.ShouldBindJSON(&req); err != nil {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "请求参数格式错误: "+err.Error())
		return
	}

	if req.Name == "" || req.Type == "" || req.Value == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "记录名称、类型和值不能为空")
		return
	}

	err := d.dns.AddRecord(c, domainName, req)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_OK(c)
}

// UpdateRecord 更新解析记录
func (d *DNS) UpdateRecord(c *gin.Context) {
	domainName := c.Param("domain")
	recordName := c.Param("record")
	if domainName == "" || recordName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名和记录名称参数不能为空")
		return
	}

	var req core.Record
	if err := c.ShouldBindJSON(&req); err != nil {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "请求参数格式错误: "+err.Error())
		return
	}

	err := d.dns.UpdateRecord(c, domainName, recordName, req)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_OK(c)
}

// DeleteRecord 删除解析记录
func (d *DNS) DeleteRecord(c *gin.Context) {
	domainName := c.Param("domain")
	recordName := c.Param("record")
	if domainName == "" || recordName == "" {
		d.svcCtx.RESP.RESP_PARAMS_ERROR(c, "域名和记录名称参数不能为空")
		return
	}

	err := d.dns.DeleteRecord(c, domainName, recordName)
	if err != nil {
		d.svcCtx.RESP.RESP_ERROR(c, http.StatusInternalServerError, err.Error())
		return
	}

	d.svcCtx.RESP.RESP_OK(c)
}
