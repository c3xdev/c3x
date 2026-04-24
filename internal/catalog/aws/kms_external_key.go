package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type KMSExternalKey struct {
	Address string
	Region  string
}

func (r *KMSExternalKey) CoreType() string {
	return "KMSExternalKey"
}

func (r *KMSExternalKey) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *KMSExternalKey) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KMSExternalKey) BuildResource() *engine.Estimate {
	kmsKey := &KMSKey{
		Region: r.Region,
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{kmsKey.customerMasterKeyCostComponent()},
		UsageSchema:    r.UsageSchema(),
	}
}
