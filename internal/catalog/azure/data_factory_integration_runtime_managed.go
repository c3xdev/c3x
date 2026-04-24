package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// DataFactoryIntegrationRuntimeManaged struct represents Data Factory's Managed
// VNET integration runtime.
//
// Resource information: https://azure.microsoft.com/en-us/services/data-factory/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/data-factory/data-pipeline/
type DataFactoryIntegrationRuntimeManaged struct {
	Address string
	Region  string

	Instances       int64
	InstanceType    string
	Enterprise      bool
	LicenseIncluded bool

	// "usage" args
	MonthlyOrchestrationRuns *int64 `c3x_usage:"monthly_orchestration_runs"`
}

func (r *DataFactoryIntegrationRuntimeManaged) CoreType() string {
	return "DataFactoryIntegrationRuntimeManaged"
}

func (r *DataFactoryIntegrationRuntimeManaged) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_orchestration_runs", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the DataFactoryIntegrationRuntimeManaged.
// It uses the `c3x_usage` struct tags to populate data into the DataFactoryIntegrationRuntimeManaged.
func (r *DataFactoryIntegrationRuntimeManaged) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid DataFactoryIntegrationRuntimeManaged struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *DataFactoryIntegrationRuntimeManaged) BuildResource() *engine.Estimate {
	runtimeFilter := "Azure Managed VNET"

	// SSIS and Managed runtime resources share the same compute configuration.
	// Terraform provider has deprecated Managed VNET runtime resource in favor of
	// SSIS one.
	ssis := DataFactoryIntegrationRuntimeAzureSSIS{
		Address:         r.Address,
		Region:          r.Region,
		Enterprise:      r.Enterprise,
		LicenseIncluded: r.LicenseIncluded,
		Instances:       r.Instances,
		InstanceType:    r.InstanceType,
	}

	costComponents := []*engine.LineItem{
		ssis.computeCostComponent(),
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
