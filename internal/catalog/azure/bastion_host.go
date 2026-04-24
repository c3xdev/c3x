package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type BastionHost struct {
	Address               string
	Region                string
	MonthlyOutboundDataGB *float64 `c3x_usage:"monthly_outbound_data_gb"`
}

func (r *BastionHost) CoreType() string {
	return "BastionHost"
}

func (r *BastionHost) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_outbound_data_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *BastionHost) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *BastionHost) BuildResource() *engine.Estimate {
	productType := "Basic"

	costComponents := []*engine.LineItem{
		{
			Name:           "Bastion host",
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Azure Bastion"),
				ProductFamily: strPtr("Networking"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", Value: strPtr(productType)},
					{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Gateway", productType))},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}

	costComponents = append(costComponents, r.outboundDataTransferComponents(productType)...)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *BastionHost) outboundDataTransferComponents(productType string) []*engine.LineItem {
	costComponents := make([]*engine.LineItem, 0)
	name := "Outbound data transfer "

	type dataTier struct {
		name       string
		startUsage string
	}

	data := []dataTier{
		{name: fmt.Sprintf("%s%s", name, "(first 10TB)"), startUsage: "5"},
		{name: fmt.Sprintf("%s%s", name, "(next 40TB)"), startUsage: "10240"},
		{name: fmt.Sprintf("%s%s", name, "(next 100TB)"), startUsage: "51200"},
		{name: fmt.Sprintf("%s%s", name, "(next 350TB)"), startUsage: "153600"},
		{name: fmt.Sprintf("%s%s", name, "(over 500TB)"), startUsage: "512000"},
	}

	if r.MonthlyOutboundDataGB != nil {
		tierLimits := []int{10000, 50000, 150000, 500000}
		monthlyOutboundDataGb := decimal.NewFromFloat(*r.MonthlyOutboundDataGB)
		tiers := usage.CalculateTierBuckets(monthlyOutboundDataGb, tierLimits)
		for i, d := range data {
			if i < len(tiers) && tiers[i].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, r.outboundDataTransferSingleComponent(
					d.name,
					productType,
					d.startUsage,
					decimalPtr(tiers[i])))
			}
		}
	} else {
		costComponents = append(costComponents, r.outboundDataTransferSingleComponent(
			data[0].name,
			productType,
			data[0].startUsage,
			nil))
	}
	return costComponents
}

func (r *BastionHost) outboundDataTransferSingleComponent(name, productType, startUsage string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Bastion"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(productType)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Data Transfer Out", productType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr(startUsage),
		},
		UsageBased: true,
	}
}
