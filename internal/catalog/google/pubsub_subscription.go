package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type PubSubSubscription struct {
	Address              string
	MonthlyMessageDataTB *float64 `c3x_usage:"monthly_message_data_tb"`
	StorageGB            *float64 `c3x_usage:"storage_gb"`
	SnapshotStorageGB    *float64 `c3x_usage:"snapshot_storage_gb"`
}

func (r *PubSubSubscription) CoreType() string {
	return "PubSubSubscription"
}

func (r *PubSubSubscription) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_message_data_tb", ValueType: engine.Float64, DefaultValue: 0.0},
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "snapshot_storage_gb", ValueType: engine.Float64, DefaultValue: 0.0},
	}
}

func (r *PubSubSubscription) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *PubSubSubscription) BuildResource() *engine.Estimate {
	var messageDataTB, storageGB, snapshotStorageGB *decimal.Decimal

	if r != nil {
		if r.MonthlyMessageDataTB != nil {
			messageDataTB = decimalPtr(decimal.NewFromFloat(*r.MonthlyMessageDataTB))
		}
		if r.StorageGB != nil {
			storageGB = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
		}
		if r.SnapshotStorageGB != nil {
			snapshotStorageGB = decimalPtr(decimal.NewFromFloat(*r.SnapshotStorageGB))
		}
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Message delivery data",
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
			{
				Name:            "Retained acknowledged message storage",
				Unit:            "GiB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: storageGB,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud Pub/Sub"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr("Subscriptions retained acknowledged messages")},
					},
				},
				PriceFilter: &engine.RateSelector{
					EndUsageAmount: strPtr(""),
				},
				UsageBased: true,
			},
			{
				Name:            "Snapshot message backlog storage",
				Unit:            "GiB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: snapshotStorageGB,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud Pub/Sub"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr("Snapshots message backlog")},
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
