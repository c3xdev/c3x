package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

type ComputeVPNGateway struct {
	Address string
	Region  string

	MonthlyEgressDataTransferGB *ComputeVPNGatewayNetworkEgressUsage `c3x_usage:"monthly_egress_data_transfer_gb"`
}

func (r *ComputeVPNGateway) CoreType() string {
	return "ComputeVPNGateway"
}

func (r *ComputeVPNGateway) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{
			Key:          "monthly_egress_data_transfer_gb",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "monthly_egress_data_transfer_gb", Items: ComputeVPNGatewayNetworkEgressUsageSchema},
		},
	}
}

func (r *ComputeVPNGateway) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeVPNGateway) BuildResource() *engine.Estimate {
	if r.MonthlyEgressDataTransferGB == nil {
		r.MonthlyEgressDataTransferGB = &ComputeVPNGatewayNetworkEgressUsage{}
	}
	region := r.Region
	r.MonthlyEgressDataTransferGB.Region = region
	r.MonthlyEgressDataTransferGB.Address = "Network egress"
	r.MonthlyEgressDataTransferGB.PrefixName = "IPSec traffic"
	return &engine.Estimate{
		Name: r.Address,
		SubResources: []*engine.Estimate{
			r.MonthlyEgressDataTransferGB.BuildResource(),
		},
		UsageSchema: r.UsageSchema(),
	}
}
