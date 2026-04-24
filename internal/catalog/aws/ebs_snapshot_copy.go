package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EBSSnapshotCopy struct {
	Address string
	Region  string
	SizeGB  *float64
}

func (r *EBSSnapshotCopy) CoreType() string {
	return "EBSSnapshotCopy"
}

func (r *EBSSnapshotCopy) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EBSSnapshotCopy) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EBSSnapshotCopy) BuildResource() *engine.Estimate {
	region := r.Region

	gbVal := decimal.NewFromInt(int64(defaultVolumeSize))

	if r.SizeGB != nil {
		gbVal = decimal.NewFromFloat(*r.SizeGB)
	}

	costComponents := []*engine.LineItem{
		ebsSnapshotCostComponent(region, gbVal),
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
