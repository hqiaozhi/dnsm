package core

import (
	"dnsm/internal/conf"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSEngine 定义DNS引擎的核心接口
type IEngine interface {
	Start() error
	Stop() error
	HandleRequest(w dns.ResponseWriter, req *dns.Msg)
	FindRecord(qname string, qtype uint16) (*conf.Record, bool)
	IsDomainConfigured(qname string) bool
	ForwardRequest(req *dns.Msg) (*dns.Msg, error)
	Match(qname, rule string) bool
}

// DefaultDNSEngine 是DNSEngine接口的默认实现
type DNSEngine struct {
	conf   *conf.Config
	server *dns.Server
}

// New 创建一个新的DNSEngine实例
func New(conf *conf.Config) *DNSEngine {
	return &DNSEngine{
		conf: conf,
	}
}

// Start 实现DNSEngine接口的Start方法
func (e *DNSEngine) Start() error {
	// 确保 conf.C.Server.Host 是有效的 IP 地址或为空(默认所有接口)
	addr := ":53" // 默认监听所有接口的 53 端口
	if e.conf.Server.Host != "" {
		addr = net.JoinHostPort(e.conf.Server.Host, strconv.Itoa(e.conf.Server.Port))
	}

	e.server = &dns.Server{Addr: addr, Net: "udp"}
	dns.HandleFunc(".", e.HandleRequest) // 所有请求都由HandleRequest处理

	log.Printf("Starting DNS server on %s\n", addr)
	return e.server.ListenAndServe()
}

// Stop 实现DNSEngine接口的Stop方法
func (e *DNSEngine) Stop() error {
	if e.server != nil {
		log.Println("Stopping DNS server...")
		return e.server.Shutdown()
	}
	return nil
}

// HandleRequest 实现DNSEngine接口的HandleRequest方法
func (e *DNSEngine) HandleRequest(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(req)
	m.RecursionAvailable = true

	// 获取第一个问题
	if len(req.Question) == 0 {
		_ = w.WriteMsg(m) // 忽略写入错误
		return
	}

	question := req.Question[0]
	qname := question.Name // 如: www.muname.com.
	qtype := question.Qtype

	// 1. 首先判断请求的域名是否在本地配置范围内
	if e.IsDomainConfigured(qname) {
		// 2a. 如果在本地配置范围内，则尝试查找匹配的记录
		foundRecord := false // 标记是否找到了匹配且类型正确的记录

		// 查找匹配的记录，优先精确匹配，然后是泛解析匹配
		record, found := e.FindRecord(qname, qtype)
		if found {
			// 根据记录类型创建相应的DNS记录
			switch record.Type {
			case "A":
				if qtype == dns.TypeA {
					ip := net.ParseIP(record.Value)
					if ip != nil && ip.To4() != nil {
						rr := &dns.A{
							Hdr: dns.RR_Header{
								Name:   qname,
								Rrtype: dns.TypeA,
								Class:  dns.ClassINET,
								Ttl:    uint32(record.TTL),
							},
							A: ip.To4(),
						}
						m.Answer = append(m.Answer, rr)
						foundRecord = true
					}
				}
			case "AAAA":
				if qtype == dns.TypeAAAA {
					ip := net.ParseIP(record.Value)
					if ip != nil && ip.To16() != nil {
						rr := &dns.AAAA{
							Hdr: dns.RR_Header{
								Name:   qname,
								Rrtype: dns.TypeAAAA,
								Class:  dns.ClassINET,
								Ttl:    uint32(record.TTL),
							},
							AAAA: ip.To16(),
						}
						m.Answer = append(m.Answer, rr)
						foundRecord = true
					}
				}
			case "CNAME":
				if qtype == dns.TypeCNAME {
					// 确保 CNAME 值以 . 结尾（FQDN）
					cnameValue := record.Value
					if !strings.HasSuffix(cnameValue, ".") {
						cnameValue += "."
					}

					rr := &dns.CNAME{
						Hdr: dns.RR_Header{
							Name:   qname,
							Rrtype: dns.TypeCNAME,
							Class:  dns.ClassINET,
							Ttl:    uint32(record.TTL),
						},
						Target: cnameValue,
					}
					m.Answer = append(m.Answer, rr)
					foundRecord = true
				}
			case "TXT":
				if qtype == dns.TypeTXT {
					rr := &dns.TXT{
						Hdr: dns.RR_Header{
							Name:   qname,
							Rrtype: dns.TypeTXT,
							Class:  dns.ClassINET,
							Ttl:    uint32(record.TTL),
						},
						Txt: []string{record.Value},
					}
					m.Answer = append(m.Answer, rr)
					foundRecord = true
				}
				// 其他记录类型的处理可以在这里添加
			}
		}

		if !foundRecord {
			// 域名匹配但在本地配置中没找到对应 qtype 的记录 -> NOERROR, 空 Answer
			m.SetRcode(req, dns.RcodeSuccess)
		}

	} else {
		// 如果不在本地配置范围内，则直接转发请求
		upstreamResp, err := e.ForwardRequest(req)
		if err != nil || upstreamResp == nil {
			log.Printf("Error forwarding request for %s: %v", qname, err)
			m.SetRcode(req, dns.RcodeServerFailure)
			m.RecursionAvailable = false
		} else {
			// 直接使用上游响应的 Answer、Authority、Additional
			m.Answer = upstreamResp.Answer
			m.Ns = upstreamResp.Ns
			m.Extra = upstreamResp.Extra
			m.Rcode = upstreamResp.Rcode
		}
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Printf("Failed to write DNS response for %s: %v", qname, err)
	}
}

// FindRecord 实现DNSEngine接口的FindRecord方法
func (e *DNSEngine) FindRecord(qname string, qtype uint16) (*conf.Record, bool) {
	// 使用线程安全的方法获取域名配置
	domains := e.conf.GetDomains()
	// 遍历所有本地配置的域名
	for _, domainConfig := range domains {
		// 先查找精确匹配的记录
		for _, record := range domainConfig.Records {
			// 检查记录名是否精确匹配 qname
			recordName := strings.ToLower(strings.TrimSuffix(record.Name, "."))
			queryName := strings.ToLower(strings.TrimSuffix(qname, "."))

			if recordName == queryName { // 精确匹配
				// 检查记录类型是否匹配查询类型
				switch record.Type {
				case "A":
					if qtype == dns.TypeA {
						return &record, true
					}
				case "AAAA":
					if qtype == dns.TypeAAAA {
						return &record, true
					}
				case "CNAME":
					if qtype == dns.TypeCNAME {
						return &record, true
					}
				case "TXT":
					if qtype == dns.TypeTXT {
						return &record, true
					}
					// 其他记录类型的检查可以在这里添加
				}
			}
		}

		// 如果没有找到精确匹配，再查找泛解析匹配
		for _, record := range domainConfig.Records {
			// 检查记录名是否是泛解析并且匹配 qname
			if strings.HasPrefix(record.Name, "*") && e.Match(qname, record.Name) {
				// 检查记录类型是否匹配查询类型
				switch record.Type {
				case "A":
					if qtype == dns.TypeA {
						return &record, true
					}
				case "AAAA":
					if qtype == dns.TypeAAAA {
						return &record, true
					}
					// 其他记录类型的检查可以在这里添加
				}
			}
		}
	}

	return nil, false
}

// IsDomainConfigured 实现DNSEngine接口的IsDomainConfigured方法
func (e *DNSEngine) IsDomainConfigured(qname string) bool {
	// 使用线程安全的方法获取域名配置
	domains := e.conf.GetDomains()
	// 遍历所有本地配置的域名
	for _, domainConfig := range domains {
		for _, record := range domainConfig.Records {
			if e.Match(qname, record.Name) {
				return true // 找到匹配的记录名，也认为是本地配置的域
			}
		}
	}
	return false // 没有在本地配置中找到匹配的域
}

// DefaultDNSForwarder 是DNSForwarder接口的默认实现
type DefaultDNSForwarder struct{}

// ForwardRequest 实现DNSForwarder接口的ForwardRequest方法
func (e *DNSEngine) ForwardRequest(req *dns.Msg) (*dns.Msg, error) {
	client := &dns.Client{
		Net:          "udp",
		DialTimeout:  3 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// 使用线程安全的方法获取上游DNS服务器列表
	upstreams := e.conf.GetUpstream()
	for _, upstream := range upstreams {
		log.Printf("Attempting to forward query to upstream server: %s", upstream)

		// 复制原始请求（避免修改原 req）
		reqCopy := req.Copy()

		resp, _, err := client.Exchange(reqCopy, upstream)
		if err != nil {
			log.Printf("Failed to exchange with upstream %s: %v", upstream, err)
			continue
		}
		if resp == nil {
			log.Printf("Upstream %s returned a nil response message", upstream)
			continue
		}

		log.Printf("Successfully forwarded query to %s", upstream)
		return resp, nil
	}

	return nil, fmt.Errorf("failed to get a valid response from any of the configured upstream servers")
}

// Match 实现DomainMatcher接口的Match方法
func (e *DNSEngine) Match(qname, rule string) bool {
	// 规范化：都转小写，确保结尾有 .
	qname = strings.ToLower(strings.TrimSuffix(qname, "."))
	rule = strings.ToLower(strings.TrimSuffix(rule, "."))

	// 精确匹配
	if qname == rule {
		return true
	}

	// 通配符匹配：*.domain.com
	if strings.HasPrefix(rule, "*") {
		suffix := rule[1:] // 去掉 *
		return strings.HasSuffix(qname, suffix)
	}

	return false
}
