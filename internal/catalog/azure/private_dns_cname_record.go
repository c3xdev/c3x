package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type PrivateDNSCNameRecord struct {
	Address        string
	Region         string
	MonthlyQueries *int64 `c3x_usage:"monthly_queries"`
}

func (r *PrivateDNSCNameRecord) CoreType() string {
	return "PrivateDNSCNameRecord"
}

func (r *PrivateDNSCNameRecord) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *PrivateDNSCNameRecord) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *PrivateDNSCNameRecord) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: dnsQueriesCostComponent(r.Region, r.MonthlyQueries),
		UsageSchema:    r.UsageSchema(),
	}
}
