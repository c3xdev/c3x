package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type SNSTopicSubscription struct {
	Address         string
	Protocol        string
	Region          string
	RequestSizeKB   *float64 `c3x_usage:"request_size_kb"`
	MonthlyRequests *int64   `c3x_usage:"monthly_requests"`
}

func (r *SNSTopicSubscription) CoreType() string {
	return "SNSTopicSubscription"
}

func (r *SNSTopicSubscription) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "request_size_kb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SNSTopicSubscription) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SNSTopicSubscription) BuildResource() *engine.Estimate {
	var endpointType string
	var freeTier string
	switch r.Protocol {
	case "http", "https":
		endpointType = "HTTP"
		freeTier = "100000"
	default:
		return &engine.Estimate{
			Name:      r.Address,
			NoPrice:   true,
			IsSkipped: true,
		}
	}

	var requests *decimal.Decimal

	requestSize := decimal.NewFromInt(64)
	if r.RequestSizeKB != nil {
		requestSize = decimal.NewFromFloat(*r.RequestSizeKB)
	}

	if r.MonthlyRequests != nil {
		requests = decimalPtr(r.calculateRequests(requestSize, decimal.NewFromInt(*r.MonthlyRequests)))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            fmt.Sprintf("%s notifications", endpointType),
				Unit:            "1M notifications",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: requests,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonSNS"),
					ProductFamily: strPtr("Message Delivery"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "endpointType", ValueRegex: strPtr(fmt.Sprintf("/%s/i", endpointType))},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr(freeTier),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *SNSTopicSubscription) calculateRequests(requestSize decimal.Decimal, monthlyRequests decimal.Decimal) decimal.Decimal {
	return requestSize.Div(decimal.NewFromInt(64)).Ceil().Mul(monthlyRequests)
}
