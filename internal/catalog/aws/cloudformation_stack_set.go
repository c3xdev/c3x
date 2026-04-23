package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type CloudFormationStackSet struct {
	Address                  string
	Region                   string
	TemplateBody             string
	MonthlyHandlerOperations *int64 `c3x_usage:"monthly_handler_operations"`
	MonthlyDurationSecs      *int64 `c3x_usage:"monthly_duration_secs"`
}

func (r *CloudFormationStackSet) CoreType() string {
	return "CloudFormationStackSet"
}

func (r *CloudFormationStackSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_handler_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_duration_secs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *CloudFormationStackSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudFormationStackSet) BuildResource() *engine.Estimate {
	stack := &CloudFormationStack{
		Region:                   r.Region,
		TemplateBody:             r.TemplateBody,
		MonthlyHandlerOperations: r.MonthlyHandlerOperations,
		MonthlyDurationSecs:      r.MonthlyDurationSecs,
	}

	if stack.checkAWS() || stack.checkAlexa() || stack.checkCustom() {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: stack.costComponents(),
		UsageSchema:    r.UsageSchema(),
	}
}
