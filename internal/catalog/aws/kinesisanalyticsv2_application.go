package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type KinesisAnalyticsV2Application struct {
	Address                    string
	Region                     string
	RuntimeEnvironment         string
	KinesisProcessingUnits     *int64   `c3x_usage:"kinesis_processing_units"`
	DurableApplicationBackupGB *float64 `c3x_usage:"durable_application_backup_gb"`
}

func (r *KinesisAnalyticsV2Application) CoreType() string {
	return "KinesisAnalyticsV2Application"
}

func (r *KinesisAnalyticsV2Application) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "kinesis_processing_units", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "durable_application_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *KinesisAnalyticsV2Application) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KinesisAnalyticsV2Application) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	var kinesisProcessingUnits *decimal.Decimal
	if r.KinesisProcessingUnits != nil {
		kinesisProcessingUnits = decimalPtr(decimal.NewFromInt(*r.KinesisProcessingUnits))
	}

	var durableApplicationBackupGB *decimal.Decimal
	if r.DurableApplicationBackupGB != nil {
		durableApplicationBackupGB = decimalPtr(decimal.NewFromFloat(*r.DurableApplicationBackupGB))
	}

	v1App := &KinesisAnalyticsApplication{
		Region:                 r.Region,
		KinesisProcessingUnits: r.KinesisProcessingUnits,
	}

	costComponents = append(costComponents, v1App.processingStreamCostComponent(kinesisProcessingUnits))

	if strings.HasPrefix(strings.ToLower(r.RuntimeEnvironment), "flink") {
		costComponents = append(costComponents, r.processingOrchestrationCostComponent())
		costComponents = append(costComponents, r.runningStorageCostComponent(kinesisProcessingUnits))
		costComponents = append(costComponents, r.backupCostComponent(durableApplicationBackupGB))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *KinesisAnalyticsV2Application) processingOrchestrationCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Processing (orchestration)",
		Unit:           "KPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisAnalytics"),
			ProductFamily: strPtr("Kinesis Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/KPU-Hour-Java/i")},
			},
		},
	}
}

func (r *KinesisAnalyticsV2Application) runningStorageCostComponent(kinesisProcessingUnits *decimal.Decimal) *engine.LineItem {
	var quantity *decimal.Decimal
	if kinesisProcessingUnits != nil {
		quantity = decimalPtr(kinesisProcessingUnits.Mul(decimal.NewFromInt(50)))
	}

	return &engine.LineItem{
		Name:            "Running storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisAnalytics"),
			ProductFamily: strPtr("Kinesis Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/RunningApplicationStorage$/i")},
			},
		},
		UsageBased: true,
	}
}

func (r *KinesisAnalyticsV2Application) backupCostComponent(durableApplicationBackupGB *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Backup",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: durableApplicationBackupGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisAnalytics"),
			ProductFamily: strPtr("Kinesis Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/DurableApplicationBackups/i")},
			},
		},
		UsageBased: true,
	}
}
