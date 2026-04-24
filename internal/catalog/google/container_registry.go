package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

type ContainerRegistry struct {
	Address                     string
	Region                      string
	Location                    string
	StorageClass                string
	StorageGB                   *float64                             `c3x_usage:"storage_gb"`
	MonthlyClassAOperations     *int64                               `c3x_usage:"monthly_class_a_operations"`
	MonthlyClassBOperations     *int64                               `c3x_usage:"monthly_class_b_operations"`
	MonthlyEgressDataTransferGB *ContainerRegistryNetworkEgressUsage `c3x_usage:"monthly_egress_data_transfer_gb"`
}

func (r *ContainerRegistry) CoreType() string {
	return "ContainerRegistry"
}

func (r *ContainerRegistry) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_class_a_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_class_b_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_data_retrieval_gb", ValueType: engine.Float64, DefaultValue: 0},
		{
			Key:          "monthly_egress_data_transfer_gb",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "monthly_egress_data_transfer_gb", Items: ContainerRegistryNetworkEgressUsageSchema},
		},
	}
}

func (r *ContainerRegistry) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ContainerRegistry) BuildResource() *engine.Estimate {
	if r.MonthlyEgressDataTransferGB == nil {
		r.MonthlyEgressDataTransferGB = &ContainerRegistryNetworkEgressUsage{}
	}
	region := r.Region
	components := []*engine.LineItem{
		dataStorageCostComponent(r.Location, r.StorageClass, r.StorageGB),
	}

	components = append(components, operationsCostComponents(r.StorageClass, r.MonthlyClassAOperations, r.MonthlyClassBOperations)...)

	r.MonthlyEgressDataTransferGB.Region = region
	r.MonthlyEgressDataTransferGB.Address = "Network egress"
	r.MonthlyEgressDataTransferGB.PrefixName = "Data transfer"
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: components,
		SubResources: []*engine.Estimate{
			r.MonthlyEgressDataTransferGB.BuildResource(),
		}, UsageSchema: r.UsageSchema(),
	}
}
