package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// DataFactoryIntegrationRuntimeSelfHosted struct represents Data Factory's
// Self-hosted runtime.
//
// Resource information: https://azure.microsoft.com/en-us/services/data-factory/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/data-factory/data-pipeline/
type DataFactoryIntegrationRuntimeSelfHosted struct {
	Address string
	Region  string

	// "usage" args
	MonthlyOrchestrationRuns *int64 `c3x_usage:"monthly_orchestration_runs"`
}

func (r *DataFactoryIntegrationRuntimeSelfHosted) CoreType() string {
	return "DataFactoryIntegrationRuntimeSelfHosted"
}

func (r *DataFactoryIntegrationRuntimeSelfHosted) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_orchestration_runs", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the DataFactoryIntegrationRuntimeSelfHosted.
// It uses the `c3x_usage` struct tags to populate data into the DataFactoryIntegrationRuntimeSelfHosted.
func (r *DataFactoryIntegrationRuntimeSelfHosted) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid DataFactoryIntegrationRuntimeSelfHosted struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *DataFactoryIntegrationRuntimeSelfHosted) BuildResource() *engine.Estimate {
	runtimeFilter := "Self Hosted"

	costComponents := []*engine.LineItem{
		dataFactoryOrchestrationCostComponent(r.Region, runtimeFilter, r.MonthlyOrchestrationRuns),
		dataFactoryDataMovementCostComponent(r.Region, runtimeFilter),
		dataFactoryPipelineCostComponent(r.Region, runtimeFilter),
		dataFactoryExternalPipelineCostComponent(r.Region, runtimeFilter),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
