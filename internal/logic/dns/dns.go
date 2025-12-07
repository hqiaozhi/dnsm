package dns

import "dnsm/internal/svc"

type DNSLogic struct {
	svcCtx *svc.SvcContext
}

func New(svcCtx *svc.SvcContext) *DNSLogic {
	return &DNSLogic{
		svcCtx: svcCtx,
	}
}
