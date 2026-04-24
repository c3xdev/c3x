package google

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// ContainerCluster struct represents Container Cluster resource.
type ContainerCluster struct {
	Address string
	Region  string

	AutopilotEnabled bool

	IsZone          bool
	DefaultNodePool *ContainerNodePool
	NodePools       []*ContainerNodePool

	// "usage" args
	DefaultNodePoolNodes        *int64   `c3x_usage:"nodes"`
	AutopilotVCPUCount          *float64 `c3x_usage:"autopilot_vcpu_count"`
	AutopilotMemoryGB           *float64 `c3x_usage:"autopilot_memory_gb"`
	AutopilotEphemeralStorageGB *float64 `c3x_usage:"autopilot_ephemeral_storage_gb"`
}

func (r *ContainerCluster) CoreType() string {
	return "ContainerCluster"
}

func (r *ContainerCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "nodes", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "autopilot_vcpu_count", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "autopilot_memory_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "autopilot_ephemeral_storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ContainerCluster.
// It uses the `c3x_usage` struct tags to populate data into the ContainerCluster.
func (r *ContainerCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	if u == nil {
		return
	}

	catalog.PopulateArgsWithUsage(r, u)

	if r.DefaultNodePool != nil {
		r.DefaultNodePool.PopulateUsage(u)
	}

	for _, nodePool := range r.NodePools {
		nodePool.PopulateUsage(&engine.ConsumptionProfile{
			Attributes: u.Get(nodePool.Address).Map(),
		})
	}
}

// BuildResource builds a engine.Estimate from a valid ContainerCluster struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ContainerCluster) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.managementFeeCostComponent())

	if r.AutopilotEnabled {
		costComponents = append(costComponents, r.autopilotCPUCostComponent())
		costComponents = append(costComponents, r.autopilotMemoryCostComponent())
		costComponents = append(costComponents, r.autopilotStorageCostComponent())
	}

	subresources := []*engine.Estimate{}

	if r.DefaultNodePool != nil {
		poolResource := r.DefaultNodePool.BuildResource()
		if poolResource != nil {
			subresources = append(subresources, poolResource)
		}
	}

	for _, nodePool := range r.NodePools {
		poolResource := nodePool.BuildResource()
		if poolResource != nil {
			subresources = append(subresources, poolResource)
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
		SubResources:   subresources,
	}
}

// managementFeeCostComponent returns a cost component for cluster management
// fee.
func (r *ContainerCluster) managementFeeCostComponent() *engine.LineItem {
	description := "Regional Kubernetes Clusters"
	name := "Cluster management fee"

	if r.IsZone {
		description = "Zonal Kubernetes Clusters"
	}

	if r.AutopilotEnabled {
		description = "Autopilot Kubernetes Clusters"
		name = "Autopilot"
	}

	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr("global"),
			Service:       strPtr("Kubernetes Engine"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr(description)},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
			EndUsageAmount:   strPtr(""),
		},
	}
}

// autopilotCPUCostComponent returns a cost component for Autopilot vCPU usage.
func (r *ContainerCluster) autopilotCPUCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	multiplier := decimal.NewFromInt(1000) // Price is for mCPU

	if r.AutopilotVCPUCount != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.AutopilotVCPUCount).Mul(multiplier))
	}

	return &engine.LineItem{
		Name:           "Autopilot vCPU",
		Unit:           "vCPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier.Mul(multiplier),
		HourlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Kubernetes Engine"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("^Autopilot Pod mCPU Requests")},
			},
		},
		UsageBased: true,
	}
}

// autopilotMemoryCostComponent returns a cost component for Autopilot memory usage.
func (r *ContainerCluster) autopilotMemoryCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.AutopilotMemoryGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.AutopilotMemoryGB))
	}

	return &engine.LineItem{
		Name:           "Autopilot memory",
		Unit:           "GB",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Kubernetes Engine"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("^Autopilot Pod Memory Requests")},
			},
		},
		UsageBased: true,
	}
}

// autopilotStorageCostComponent returns a cost component for Autopilot
// ephemeral storage usage.
func (r *ContainerCluster) autopilotStorageCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.AutopilotEphemeralStorageGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.AutopilotEphemeralStorageGB))
	}

	return &engine.LineItem{
		Name:           "Autopilot ephemeral storage",
		Unit:           "GB",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Kubernetes Engine"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("^Autopilot Pod Ephemeral Storage Requests")},
			},
		},
		UsageBased: true,
	}
}
