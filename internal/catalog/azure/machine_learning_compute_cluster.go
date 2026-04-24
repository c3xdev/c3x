package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// MachineLearningComputeCluster struct represents a Azure Machine Learning Compute Cluster.
//
// These use the same pricing as Azure Linux Virtual Machines. We default to the minimum scale of
// the cluster, but allow the number of instances and monthly hours of each instance to be set.
//
// Resource information: https://azure.microsoft.com/en-gb/pricing/details/machine-learning/#overview
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/machine-learning/
type MachineLearningComputeCluster struct {
	Address      string
	Region       string
	InstanceType string
	MinNodeCount int64
	Instances    *int64   `c3x_usage:"instances"`
	MonthlyHours *float64 `c3x_usage:"monthly_hrs"`
}

// CoreType returns the name of this resource type
func (r *MachineLearningComputeCluster) CoreType() string {
	return "MachineLearningComputeCluster"
}

// UsageSchema defines a list which represents the usage schema of MachineLearningComputeCluster.
func (r *MachineLearningComputeCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_hrs", ValueType: engine.Float64, DefaultValue: 0},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the MachineLearningComputeCluster.
// It uses the `c3x_usage` struct tags to populate data into the MachineLearningComputeCluster.
func (r *MachineLearningComputeCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid MachineLearningComputeCluster struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *MachineLearningComputeCluster) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		linuxVirtualMachineCostComponent(r.Region, r.InstanceType, r.MonthlyHours),
	}

	res := &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}

	instances := r.MinNodeCount

	// If the user has set the monthly hours, but the min node count is 0,
	// we assume that the user wants to calculate the cost of 1 instance.
	if r.MonthlyHours != nil && instances == 0 {
		instances = 1
	}

	if r.Instances != nil {
		instances = *r.Instances
	}

	engine.ScaleQuantities(res, decimal.NewFromInt(instances))

	return res
}
