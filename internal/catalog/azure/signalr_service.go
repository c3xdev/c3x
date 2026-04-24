package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// SignalRService struct represents an Azure SignalR Service.
//
// Resource information: https://azure.microsoft.com/en-us/products/signalr-service
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/signalr-service/
type SignalRService struct {
	Address     string
	Region      string
	SkuName     string
	SkuCapacity int64

	MonthlyAdditionalMessages *int64 `c3x_usage:"monthly_additional_messages"`
}

// CoreType returns the name of this resource type
func (r *SignalRService) CoreType() string {
	return "SignalRService"
}

// UsageSchema defines a list which represents the usage schema of SignalRService.
func (r *SignalRService) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_additional_messages", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the SignalRService.
// It uses the `c3x_usage` struct tags to populate data into the SignalRService.
func (r *SignalRService) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid SignalRService struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *SignalRService) BuildResource() *engine.Estimate {
	// normalize sku to first letter capitalized
	sku := cases.Title(language.English).String(strings.ToLower(r.SkuName))

	if s := strings.Split(r.SkuName, "_"); len(s) == 2 {
		sku = s[0]
	}

	if sku == "Free" {
		return &engine.Estimate{
			Name:      r.Address,
			IsSkipped: true,
			NoPrice:   true,
		}
	}

	costComponents := []*engine.LineItem{
		r.serviceUsageCostComponent(sku, r.SkuCapacity),
		r.additionalMessagesCostComponent(sku, r.MonthlyAdditionalMessages),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *SignalRService) serviceUsageCostComponent(sku string, capacity int64) *engine.LineItem {
	return &engine.LineItem{
		Name: fmt.Sprintf("Service usage (%s)", sku),
		Unit: "units",
		// This is a bit of a hack, but the Azure pricing API returns the price per day,
		// so we need to convert the price per day to price per hour.
		UnitMultiplier:  engine.DaysInMonth,
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(capacity).Mul(engine.DaysInMonth)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("SignalR"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Unit", sku))},
			},
		},
	}
}

func (r *SignalRService) additionalMessagesCostComponent(sku string, quantity *int64) *engine.LineItem {
	var q *decimal.Decimal
	if quantity != nil {
		q = decimalPtr(decimal.NewFromInt(*quantity).Div(decimal.NewFromInt(1000000)))
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("Additional messages (%s)", sku),
		Unit:            "1M messages",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: q,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("SignalR"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Message", sku))},
			},
		},
		UsageBased: true,
	}
}
