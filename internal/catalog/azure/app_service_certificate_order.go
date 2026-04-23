package azure

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type AppServiceCertificateOrder struct {
	Address     string
	ProductType string
}

func (r *AppServiceCertificateOrder) CoreType() string {
	return "AppServiceCertificateOrder"
}

func (r *AppServiceCertificateOrder) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *AppServiceCertificateOrder) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AppServiceCertificateOrder) BuildResource() *engine.Estimate {
	region := "Global"

	if strings.HasPrefix(region, "usgov") {
		region = "US Gov"
	}

	productType := "Standard"
	if r.ProductType != "" {
		productType = r.ProductType
	}
	productType = strings.ToLower(productType)

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("SSL certificate (%s)", productType),
			Unit:           "years",
			UnitMultiplier: decimal.NewFromInt(1),

			MonthlyQuantity: decimalPtr(decimal.NewFromInt(1).Div(decimal.NewFromInt(12))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Azure App Service"),
				ProductFamily: strPtr("Compute"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/%s SSL - 1 Year/i", productType))},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
