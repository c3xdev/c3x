package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type Route53HealthCheck struct {
	Address         string
	RequestInterval string
	MeasureLatency  bool
	Type            string
	EndpointType    *string `c3x_usage:"endpoint_type"`
}

func (r *Route53HealthCheck) CoreType() string {
	return "Route53HealthCheck"
}

func (r *Route53HealthCheck) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "endpoint_type", ValueType: engine.String, DefaultValue: "aws"},
	}
}

func (r *Route53HealthCheck) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Route53HealthCheck) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	endpointType := "aws"
	usageAmount := "50"
	if r.EndpointType != nil {
		endpointType = strings.Replace(*r.EndpointType, "_", "-", 1)
		if strings.ToLower(endpointType) == "non-aws" {
			usageAmount = "0"
		}
	}

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Health check",
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Service:       strPtr("AmazonRoute53"),
			ProductFamily: strPtr("DNS Health Check"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/Health-Check-%s/i", endpointType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageAmount),
		},
	})

	healthCheckType := r.Type
	optionalHealthCheckCount := decimal.Zero

	if strings.HasPrefix(healthCheckType, "HTTPS") {
		optionalHealthCheckCount = optionalHealthCheckCount.Add(decimal.NewFromInt(1))
	}

	if r.RequestInterval == "10" {
		optionalHealthCheckCount = optionalHealthCheckCount.Add(decimal.NewFromInt(1))
	}

	if r.MeasureLatency {
		optionalHealthCheckCount = optionalHealthCheckCount.Add(decimal.NewFromInt(1))
	}

	if strings.HasSuffix(healthCheckType, "STR_MATCH") {
		optionalHealthCheckCount = optionalHealthCheckCount.Add(decimal.NewFromInt(1))
	}

	if optionalHealthCheckCount.GreaterThan(decimal.NewFromInt(0)) {
		costComponents = append(costComponents, &engine.LineItem{
			Name:            "Optional features",
			Unit:            "months",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(optionalHealthCheckCount),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Service:       strPtr("AmazonRoute53"),
				ProductFamily: strPtr("DNS Health Check"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/Health-Check-Option-%s/i", endpointType))},
				},
			},
			PriceFilter: &engine.RateSelector{
				StartUsageAmount: strPtr("0"),
			},
		})
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
