package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type DNSNSRecord struct {
	Address        string
	Region         string
	MonthlyQueries *int64 `c3x_usage:"monthly_queries"`
}

func (r *DNSNSRecord) CoreType() string {
	return "DNSNSRecord"
}

func (r *DNSNSRecord) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *DNSNSRecord) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DNSNSRecord) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: dnsQueriesCostComponent(r.Region, r.MonthlyQueries),
		UsageSchema:    r.UsageSchema(),
	}
}
