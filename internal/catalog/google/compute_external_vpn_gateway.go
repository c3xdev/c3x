package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

type ComputeExternalVPNGateway struct {
	Address string
	Region  string

	MonthlyEgressDataTransferGB *ComputeExternalVPNGatewayNetworkEgressUsage `c3x_usage:"monthly_egress_data_transfer_gb"`
}

func (r *ComputeExternalVPNGateway) CoreType() string {
	return "ComputeExternalVPNGateway"
}

func (r *ComputeExternalVPNGateway) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{
			Key:       "monthly_egress_data_transfer_gb",
			ValueType: engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "monthly_egress_data_transfer_gb",
				Items: ComputeExternalVPNGatewayNetworkEgressUsageSchema},
		},
	}
}

func (r *ComputeExternalVPNGateway) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
	if r.MonthlyEgressDataTransferGB == nil {
		r.MonthlyEgressDataTransferGB = &ComputeExternalVPNGatewayNetworkEgressUsage{}
	}
}

func (r *ComputeExternalVPNGateway) BuildResource() *engine.Estimate {
	region := r.Region
	r.MonthlyEgressDataTransferGB.Region = region
	r.MonthlyEgressDataTransferGB.Address = "Network egress"
	r.MonthlyEgressDataTransferGB.PrefixName = "IPSec traffic"
	return &engine.Estimate{
		Name: r.Address,
		SubResources: []*engine.Estimate{
			r.MonthlyEgressDataTransferGB.BuildResource(),
		}, UsageSchema: r.UsageSchema(),
	}
}
