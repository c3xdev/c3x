package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// MachineLearningComputeInstance struct represents a Azure Machine Learning Compute Instance.
//
// These use the same pricing as Azure Linux Virtual Machines.
//
// Resource information: https://azure.microsoft.com/en-gb/pricing/details/machine-learning/#overview
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/machine-learning/
type MachineLearningComputeInstance struct {
	Address      string
	Region       string
	InstanceType string
	MonthlyHours *float64 `c3x_usage:"monthly_hrs"`
}

// CoreType returns the name of this resource type
func (r *MachineLearningComputeInstance) CoreType() string {
	return "MachineLearningComputeInstance"
}

// UsageSchema defines a list which represents the usage schema of MachineLearningComputeInstance.
func (r *MachineLearningComputeInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", ValueType: engine.Float64, DefaultValue: 0},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the MachineLearningComputeInstance.
// It uses the `c3x_usage` struct tags to populate data into the MachineLearningComputeInstance.
func (r *MachineLearningComputeInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid MachineLearningComputeInstance struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *MachineLearningComputeInstance) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		linuxVirtualMachineCostComponent(r.Region, r.InstanceType, r.MonthlyHours),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
