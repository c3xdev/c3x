package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type APIGatewayRestAPI struct {
	Address         string
	Region          string
	MonthlyRequests *int64 `c3x_usage:"monthly_requests"`
}

func (r *APIGatewayRestAPI) CoreType() string {
	return "APIGatewayRestAPI"
}

func (r *APIGatewayRestAPI) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *APIGatewayRestAPI) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *APIGatewayRestAPI) BuildResource() *engine.Estimate {
	var costComponents []*engine.LineItem
	var monthlyRequests *decimal.Decimal

	if r.MonthlyRequests != nil {
		monthlyRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyRequests))

		requestLimits := []int{333000000, 667000000, 19000000000}
		apiRequestQuantities := usage.CalculateTierBuckets(*monthlyRequests, requestLimits)

		costComponents = append(costComponents, r.requestsCostComponent("Requests (first 333M)", "0", &apiRequestQuantities[0]))

		if apiRequestQuantities[1].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.requestsCostComponent("Requests (next 667M)", "333000000", &apiRequestQuantities[1]))
		}

		if apiRequestQuantities[2].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.requestsCostComponent("Requests (next 19B)", "1000000000", &apiRequestQuantities[2]))
		}

		if apiRequestQuantities[3].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.requestsCostComponent("Requests (over 20B)", "20000000000", &apiRequestQuantities[3]))
		}
	} else {
		costComponents = append(costComponents, r.requestsCostComponent("Requests (first 333M)", "0", monthlyRequests))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *APIGatewayRestAPI) requestsCostComponent(displayName string, usageTier string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            displayName,
		Unit:            "1M requests",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonApiGateway"),
			ProductFamily: strPtr("API Calls"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/ApiGatewayRequest/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}
