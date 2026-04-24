package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// VPNGatewayConnection represents a VPN Gateway connection, which is a billable component
// of a S2S VPN gateway. See VPNGateway for more information.
//
// More resource information here: https://docs.microsoft.com/en-us/azure/virtual-wan/virtual-wan-about
// Pricing information here: https://azure.microsoft.com/en-us/pricing/details/virtual-wan/
type VPNGatewayConnection struct {
	// Address is the unique name of the resource in the IAC language.
	Address string
	// Region is the azure region the VPNGatewayConnection is provisioned within.
	Region string
}

func (r *VPNGatewayConnection) CoreType() string {
	return "VPNGatewayConnection"
}

func (r *VPNGatewayConnection) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the VPNGatewayConnection.
// It uses the `c3x_usage` struct tags to populate data into the VPNGatewayConnection.
func (r *VPNGatewayConnection) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid VPNGatewayConnection.
// It returns a VPNGatewayConnection as a engine.Estimate with a single cost component representing the
// connection unit. The hourly quantity is set to 1 as VPNGatewayConnection represents a single connection unit.
//
// This method is called after the resource is initialised by an iac provider.
// See providers folder for more information.
func (r *VPNGatewayConnection) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:           "S2S Connections",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Virtual WAN"),
					ProductFamily: strPtr("Networking"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "skuName", Value: strPtr("VPN S2S Connection Unit")},
					},
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("Consumption"),
				},
			},
		},
	}
}
