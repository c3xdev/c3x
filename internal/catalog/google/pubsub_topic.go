package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type PubSubTopic struct {
	Address              string
	MonthlyMessageDataTB *float64 `c3x_usage:"monthly_message_data_tb"`
}

func (r *PubSubTopic) CoreType() string {
	return "PubSubTopic"
}

func (r *PubSubTopic) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_message_data_tb", ValueType: engine.Float64, DefaultValue: 0.0},
	}
}

func (r *PubSubTopic) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *PubSubTopic) BuildResource() *engine.Estimate {
	var messageDataTB *decimal.Decimal

	if r.MonthlyMessageDataTB != nil {
		messageDataTB = decimalPtr(decimal.NewFromFloat(*r.MonthlyMessageDataTB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Message ingestion data",
				Unit:            "TiB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: messageDataTB,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud Pub/Sub"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr("Message Delivery Basic")},
					},
				},
				PriceFilter: &engine.RateSelector{
					EndUsageAmount: strPtr(""),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
