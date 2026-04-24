package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type AppServicePlan struct {
	Address     string
	SKUSize     string
	SKUCapacity int64
	Kind        string
	Region      string
	IsDevTest   bool
}

func (r *AppServicePlan) CoreType() string {
	return "AppServicePlan"
}

func (r *AppServicePlan) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *AppServicePlan) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AppServicePlan) BuildResource() *engine.Estimate {
	sku := ""
	os := "windows"
	var capacity int64 = 1
	if r.SKUCapacity > 0 {
		capacity = r.SKUCapacity
	}
	productName := "Standard Plan"

	if len(r.SKUSize) < 2 || strings.ToLower(r.SKUSize[:2]) == "ep" || strings.ToLower(r.SKUSize[:2]) == "y1" || strings.ToLower(r.SKUSize[:2]) == "ws" {
		return &engine.Estimate{
			Name:        r.Address,
			IsSkipped:   true,
			NoPrice:     true,
			UsageSchema: r.UsageSchema(),
		}
	}

	var additionalAttributeFilters []*engine.AttributeMatch

	switch strings.ToLower(r.SKUSize[:1]) {
	case "s":
		sku = "S" + r.SKUSize[1:]
	case "b":
		sku = "B" + r.SKUSize[1:]
		productName = "Basic Plan"
	case "p", "i":
		sku, productName, additionalAttributeFilters = getVersionedAppServicePlanSKU(r.SKUSize, os)
	}

	switch strings.ToLower(r.SKUSize[:2]) {
	case "pc":
		sku = "PC" + r.SKUSize[2:]
		productName = "Premium Windows Container Plan"
	case "y1":
		sku = "Shared"
		productName = "Shared Plan"
	}

	if r.Kind != "" {
		os = strings.ToLower(r.Kind)
	}
	if os == "app" {
		os = "windows"
	}
	if os != "windows" && productName != "Premium Plan" && productName != "Isolated Plan" {
		productName += " - Linux"
	}

	purchaseOption := "Consumption"
	name := fmt.Sprintf("Instance usage (%s)", r.SKUSize)
	if r.IsDevTest && strings.Contains(os, "windows") && strings.ToLower(r.SKUSize[:2]) != "pc" {
		purchaseOption = "DevTestConsumption"
		name = fmt.Sprintf("Instance usage (dev/test, %s)", r.SKUSize)
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			servicePlanCostComponent(
				r.Region,
				name,
				productName,
				sku,
				capacity,
				purchaseOption,
				additionalAttributeFilters...,
			),
		},
		UsageSchema: r.UsageSchema(),
	}
}

func servicePlanCostComponent(region, name, productName, skuRefactor string, capacity int64, purchaseOption string, additionalAttributeFilters ...*engine.AttributeMatch) *engine.LineItem {
	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(capacity)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure App Service"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: append([]*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure App Service " + productName)},
				{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/%s$/i", skuRefactor))},
			}, additionalAttributeFilters...),
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr(purchaseOption),
		},
	}
}

func getVersionedAppServicePlanSKU(skuName, os string) (string, string, []*engine.AttributeMatch) {
	tier := "Premium"
	if strings.ToLower(skuName[:1]) == "i" {
		tier = "Isolated"
	}

	version := strings.ToLower(skuName[2:])
	if version == "v1" {
		version = ""
	}

	formattedSku := strings.TrimSpace(skuName[:2] + " ?" + version)

	productVersion := version
	if len(version) > 0 && version[0] == 'm' {
		productVersion = version[1:]
	}
	productName := strings.ReplaceAll(tier+" "+productVersion+" Plan", "  ", " ")

	if productVersion == "v3" && os == "linux" {
		return formattedSku, productName, []*engine.AttributeMatch{
			{
				Key:        "armSkuName",
				ValueRegex: strPtr(fmt.Sprintf("/%s$/i", strings.ReplaceAll(formattedSku, " ", "_"))),
			},
		}
	}

	return formattedSku, productName, nil
}
