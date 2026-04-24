package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// VirtualNetworkPeering struct represents a VNET peering.
//

// Resource information: https://azure.microsoft.com/en-us/services/virtual-network/#overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/virtual-network/
type VirtualNetworkPeering struct {
	Address           string
	SourceRegion      string
	DestinationRegion string
	SourceZone        string
	DestinationZone   string

	MonthlyDataTransferGB *float64 `c3x_usage:"monthly_data_transfer_gb"`
}

func (r *VirtualNetworkPeering) CoreType() string {
	return "VirtualNetworkPeering"
}

func (r *VirtualNetworkPeering) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_transfer_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

func (r *VirtualNetworkPeering) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *VirtualNetworkPeering) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.ingressDataProcessedCostComponent(),
		r.egressDataProcessedCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *VirtualNetworkPeering) egressDataProcessedCostComponent() *engine.LineItem {
	if r.DestinationRegion == r.SourceRegion {
		return &engine.LineItem{
			Name:            "Outbound data transfer",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataTransferGB),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr("Global"),
				Service:       strPtr("Virtual Network"),
				ProductFamily: strPtr("Networking"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "meterName", Value: strPtr("Intra-Region Egress")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
			UsageBased: true,
		}
	}

	return &engine.LineItem{
		Name:            "Outbound data transfer",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataTransferGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(r.SourceZone),
			Service:    strPtr("VPN Gateway"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "serviceFamily", ValueRegex: regexPtr("Networking")},
				{Key: "productName", ValueRegex: regexPtr("VPN Gateway Bandwidth")},
				{Key: "meterName", ValueRegex: regexPtr("Inter-Virtual Network Data Transfer Out")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}

func (r *VirtualNetworkPeering) ingressDataProcessedCostComponent() *engine.LineItem {
	if r.DestinationRegion == r.SourceRegion {
		return &engine.LineItem{
			Name:            "Inbound data transfer",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataTransferGB),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr("Global"),
				Service:       strPtr("Virtual Network"),
				ProductFamily: strPtr("Networking"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "meterName", Value: strPtr("Intra-Region Ingress")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
			UsageBased: true,
		}
	}

	return &engine.LineItem{
		Name:            "Inbound data transfer",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataTransferGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(r.DestinationZone),
			Service:    strPtr("VPN Gateway"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "serviceFamily", ValueRegex: regexPtr("Networking")},
				{Key: "productName", ValueRegex: regexPtr("VPN Gateway Bandwidth")},
				{Key: "meterName", ValueRegex: regexPtr("Inter-Virtual Network Data Transfer Out")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}
