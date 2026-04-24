package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type ActiveDirectoryDomainService struct {
	Address string
	Region  string
	SKU     string
}

func (r *ActiveDirectoryDomainService) CoreType() string {
	return "ActiveDirectoryDomainService"
}

func (r *ActiveDirectoryDomainService) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ActiveDirectoryDomainService) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ActiveDirectoryDomainService) BuildResource() *engine.Estimate {
	costComponents := activeDirectoryDomainServiceCostComponents("Active directory domain service", r.Region, r.SKU)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func activeDirectoryDomainServiceCostComponents(name, region, sku string) []*engine.LineItem {
	productType := "Standard"

	if sku != "" {
		productType = sku
	}

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("%s (%s)", name, productType),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Microsoft Entra Domain Services"),
				ProductFamily: strPtr("Security"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", Value: strPtr(productType)},
					{Key: "meterName", Value: strPtr(fmt.Sprintf("%s User Forest", productType))},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}
	return costComponents
}
