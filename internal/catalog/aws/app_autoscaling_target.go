package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type AppAutoscalingTarget struct {
	Address string
	Region  string

	ResourceID        string
	ScalableDimension string

	MinCapacity int64
	MaxCapacity int64

	// "usage" args
	Capacity *int64 `c3x_usage:"capacity"`
}

func (r *AppAutoscalingTarget) CoreType() string {
	return "AppAutoscalingTarget"
}

func (r *AppAutoscalingTarget) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "capacity", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *AppAutoscalingTarget) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AppAutoscalingTarget) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
	}
}
