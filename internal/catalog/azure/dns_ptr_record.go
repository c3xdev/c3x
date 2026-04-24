package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type DNSPtrRecord struct {
	Address        string
	Region         string
	MonthlyQueries *int64 `c3x_usage:"monthly_queries"`
}

func (r *DNSPtrRecord) CoreType() string {
	return "DNSPtrRecord"
}

func (r *DNSPtrRecord) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *DNSPtrRecord) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DNSPtrRecord) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: dnsQueriesCostComponent(r.Region, r.MonthlyQueries),
		UsageSchema:    r.UsageSchema(),
	}
}
