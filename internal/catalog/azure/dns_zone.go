package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type DNSZone struct {
	Address string
	Region  string
}

func (r *DNSZone) CoreType() string {
	return "DNSZone"
}

func (r *DNSZone) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *DNSZone) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DNSZone) BuildResource() *engine.Estimate {

	var region string
	if strings.HasPrefix(strings.ToLower(r.Region), "usgov") {
		region = "US Gov Zone 1"
	} else if strings.HasPrefix(strings.ToLower(r.Region), "germany") {
		region = "DE Zone 1"
	} else if strings.HasPrefix(strings.ToLower(r.Region), "china") {
		region = "Zone 1 (China)"
	} else {
		region = "Zone 1"
	}

	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, hostedPublicZoneCostComponent(region))
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func hostedPublicZoneCostComponent(region string) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Hosted zone",
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure DNS"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("Public Zone(s)?")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
	}
}
