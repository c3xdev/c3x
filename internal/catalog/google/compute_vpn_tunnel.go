package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ComputeVPNTunnel struct {
	Address string
	Region  string
}

func (r *ComputeVPNTunnel) CoreType() string {
	return "ComputeVPNTunnel"
}

func (r *ComputeVPNTunnel) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ComputeVPNTunnel) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeVPNTunnel) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			r.vpnTunnelCostComponent(),
		}, UsageSchema: r.UsageSchema(),
	}
}

func (r *ComputeVPNTunnel) vpnTunnelCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "VPN Tunnel",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "resourceGroup", Value: strPtr("VPNTunnel")},
			},
		},
	}
}
