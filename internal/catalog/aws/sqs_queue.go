package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type SQSQueue struct {
	Address         string
	Region          string
	FifoQueue       bool
	MonthlyRequests *float64 `c3x_usage:"monthly_requests"`
	RequestSizeKB   *int64   `c3x_usage:"request_size_kb"`
}

func (r *SQSQueue) CoreType() string {
	return "SQSQueue"
}

func (r *SQSQueue) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "request_size_kb", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SQSQueue) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SQSQueue) BuildResource() *engine.Estimate {
	var queueType string
	if r.FifoQueue {
		queueType = "FIFO (first-in, first-out)"
	} else {
		queueType = "Standard"
	}

	var requests *decimal.Decimal

	requestSize := decimal.NewFromInt(64)
	if r.RequestSizeKB != nil {
		requestSize = decimal.NewFromInt(*r.RequestSizeKB)
	}

	if r.MonthlyRequests != nil {
		requests = decimalPtr(r.calculateRequests(requestSize, decimal.NewFromFloat(*r.MonthlyRequests)))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Requests",
				Unit:            "1M requests",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: requests,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSQueueService"),
					ProductFamily: strPtr("API Request"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "queueType", Value: strPtr(queueType)},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("0"),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *SQSQueue) calculateRequests(requestSize decimal.Decimal, monthlyRequests decimal.Decimal) decimal.Decimal {
	return requestSize.Div(decimal.NewFromInt(64)).Ceil().Mul(monthlyRequests)
}
