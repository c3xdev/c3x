package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type PrivateDNSARecord struct {
	Address        string
	Region         string
	MonthlyQueries *int64 `c3x_usage:"monthly_queries"`
}

func (r *PrivateDNSARecord) CoreType() string {
	return "PrivateDNSARecord"
}

func (r *PrivateDNSARecord) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *PrivateDNSARecord) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *PrivateDNSARecord) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: dnsQueriesCostComponent(r.Region, r.MonthlyQueries),
		UsageSchema:    r.UsageSchema(),
	}
}
