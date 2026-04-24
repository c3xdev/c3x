package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type AppServiceCustomHostnameBinding struct {
	Address  string
	Region   string
	SSLState string
}

func (r *AppServiceCustomHostnameBinding) CoreType() string {
	return "AppServiceCustomHostnameBinding"
}

func (r *AppServiceCustomHostnameBinding) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *AppServiceCustomHostnameBinding) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AppServiceCustomHostnameBinding) BuildResource() *engine.Estimate {
	var sslType, sslState string

	sslState = strings.ToUpper(r.SSLState)

	if strings.HasPrefix(sslState, "IP") {
		sslType = "IP"
	} else {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	var instanceCount int64 = 1

	costComponents := []*engine.LineItem{
		{
			Name:            "IP SSL certificate",
			Unit:            "months",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(instanceCount)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Azure App Service"),
				ProductFamily: strPtr("Compute"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", Value: strPtr(fmt.Sprintf("%s SSL", sslType))},
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
