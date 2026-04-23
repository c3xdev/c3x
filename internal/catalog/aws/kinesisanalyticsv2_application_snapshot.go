package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type KinesisAnalyticsV2ApplicationSnapshot struct {
	Address                    string
	Region                     string
	DurableApplicationBackupGB *float64 `c3x_usage:"durable_application_backup_gb"`
}

func (r *KinesisAnalyticsV2ApplicationSnapshot) CoreType() string {
	return "KinesisAnalyticsV2ApplicationSnapshot"
}

func (r *KinesisAnalyticsV2ApplicationSnapshot) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "durable_application_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *KinesisAnalyticsV2ApplicationSnapshot) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KinesisAnalyticsV2ApplicationSnapshot) BuildResource() *engine.Estimate {
	var durableApplicationBackupGB *decimal.Decimal
	if r.DurableApplicationBackupGB != nil {
		durableApplicationBackupGB = decimalPtr(decimal.NewFromFloat(*r.DurableApplicationBackupGB))
	}

	v2App := &KinesisAnalyticsV2Application{
		Region:                     r.Region,
		DurableApplicationBackupGB: r.DurableApplicationBackupGB,
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{v2App.backupCostComponent(durableApplicationBackupGB)},
		UsageSchema:    r.UsageSchema(),
	}
}
