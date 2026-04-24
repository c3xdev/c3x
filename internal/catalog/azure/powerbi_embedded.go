package azure

import (
	"fmt"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// PowerBIEmbedded struct represents a Power BI Embedded resource.
//
// Resource information: https://learn.microsoft.com/en-us/power-bi/developer/embedded/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/power-bi-embedded/
type PowerBIEmbedded struct {
	Address string
	SKU     string
	Region  string
}

func (r *PowerBIEmbedded) CoreType() string {
	return "PowerBIEmbedded"
}

func (r *PowerBIEmbedded) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the PowerBIEmbedded.
// It uses the `c3x_usage` struct tags to populate data into the PowerBIEmbedded.
func (r *PowerBIEmbedded) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid PowerBIEmbedded struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *PowerBIEmbedded) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{r.instanceUsageCostComponent()},
	}
}

func (r *PowerBIEmbedded) instanceUsageCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           fmt.Sprintf("Node usage (%s)", r.SKU),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Power BI Embedded"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(r.SKU)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
