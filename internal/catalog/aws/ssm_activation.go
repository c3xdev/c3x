package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type SSMActivation struct {
	Address           string
	Region            string
	RegistrationLimit int64
	InstanceTier      *string `c3x_usage:"instance_tier"`
	Instances         *int64  `c3x_usage:"instances"`
}

func (r *SSMActivation) CoreType() string {
	return "SSMActivation"
}

func (r *SSMActivation) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "instance_tier", ValueType: engine.String, DefaultValue: "standard"},
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SSMActivation) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SSMActivation) BuildResource() *engine.Estimate {
	var instanceTier string
	if r.InstanceTier != nil {
		instanceTier = *r.InstanceTier
	} else if r.RegistrationLimit > 1000 {
		instanceTier = "Advanced"
	}

	var instanceCount *decimal.Decimal
	if r.Instances != nil {
		instanceCount = decimalPtr(decimal.NewFromInt(*r.Instances))
	}

	if strings.ToLower(instanceTier) == "advanced" {
		return &engine.Estimate{
			Name: r.Address,
			CostComponents: []*engine.LineItem{
				{
					Name:           "On-prem managed instances (advanced)",
					Unit:           "hours",
					UnitMultiplier: decimal.NewFromInt(1),
					HourlyQuantity: instanceCount,
					ProductFilter: &engine.ProductSelector{
						VendorName:    strPtr("aws"),
						Region:        strPtr(r.Region),
						Service:       strPtr("AWSSystemsManager"),
						ProductFamily: strPtr("AWS Systems Manager"),
						AttributeFilters: []*engine.AttributeMatch{
							{Key: "usagetype", ValueRegex: strPtr("/MI-AdvInstances-Hrs/")},
						},
					},
					UsageBased: true,
				},
			}, UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name:        r.Address,
		NoPrice:     true,
		IsSkipped:   true,
		UsageSchema: r.UsageSchema(),
	}
}
