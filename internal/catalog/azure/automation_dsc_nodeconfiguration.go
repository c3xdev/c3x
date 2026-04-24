package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type AutomationDSCNodeConfiguration struct {
	Address string
	Region  string

	NonAzureConfigNodeCount *int64 `c3x_usage:"non_azure_config_node_count"`
}

func (r *AutomationDSCNodeConfiguration) CoreType() string {
	return "AutomationDSCNodeConfiguration"
}

func (r *AutomationDSCNodeConfiguration) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "non_azure_config_node_count", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *AutomationDSCNodeConfiguration) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AutomationDSCNodeConfiguration) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: automationDSCNodesCostComponent(&r.Region, r.NonAzureConfigNodeCount),
		UsageSchema:    r.UsageSchema(),
	}
}
