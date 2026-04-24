package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppAutoscalingTargetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_appautoscaling_target",
		CoreRFunc: NewAppAutoscalingTargetResource,
		// This reference is used by other resources (e.g. DynamoDBTable) to generate
		// a reverse reference
		ReferenceAttributes: []string{"resource_id"},
	}
}

func NewAppAutoscalingTargetResource(d *engine.ResourceSpec) engine.CatalogItem {
	return newAppAutoscalingTarget(d, nil)

}

func newAppAutoscalingTarget(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *aws.AppAutoscalingTarget {
	r := &aws.AppAutoscalingTarget{
		Address:           d.Address,
		Region:            d.Get("region").String(),
		ResourceID:        d.Get("resource_id").String(),
		ScalableDimension: d.Get("scalable_dimension").String(),
		MinCapacity:       d.Get("min_capacity").Int(),
		MaxCapacity:       d.Get("max_capacity").Int(),
	}

	r.PopulateUsage(u)

	return r
}
