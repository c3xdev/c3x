package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type ActiveDirectoryDomainServiceReplicaSet struct {
	Address            string
	Region             string
	DomainServiceIDSKU string
}

func (r *ActiveDirectoryDomainServiceReplicaSet) CoreType() string {
	return "ActiveDirectoryDomainServiceReplicaSet"
}

func (r *ActiveDirectoryDomainServiceReplicaSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ActiveDirectoryDomainServiceReplicaSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ActiveDirectoryDomainServiceReplicaSet) BuildResource() *engine.Estimate {
	region := r.Region

	costComponents := activeDirectoryDomainServiceCostComponents("Active directory domain service replica set", region, r.DomainServiceIDSKU)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
