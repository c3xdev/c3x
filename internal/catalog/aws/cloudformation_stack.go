package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type CloudFormationStack struct {
	Address                  string
	Region                   string
	TemplateBody             string
	MonthlyHandlerOperations *int64 `c3x_usage:"monthly_handler_operations"`
	MonthlyDurationSecs      *int64 `c3x_usage:"monthly_duration_secs"`
}

func (r *CloudFormationStack) CoreType() string {
	return "CloudFormationStack"
}

func (r *CloudFormationStack) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_handler_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_duration_secs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *CloudFormationStack) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudFormationStack) BuildResource() *engine.Estimate {
	if r.checkAWS() || r.checkAlexa() || r.checkCustom() {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: r.costComponents(),
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *CloudFormationStack) costComponents() []*engine.LineItem {
	var monthlyHandlerOperations, monthlyDurationSecs *decimal.Decimal

	if r.MonthlyHandlerOperations != nil {
		monthlyHandlerOperations = decimalPtr(decimal.NewFromInt(*r.MonthlyHandlerOperations))
	}
	if r.MonthlyDurationSecs != nil {
		monthlyDurationSecs = decimalPtr(decimal.NewFromInt(*r.MonthlyDurationSecs))
	}

	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, r.cloudFormationCostComponent("Handler operations", "operations", "Resource-Invocation-Count", monthlyHandlerOperations))
	costComponents = append(costComponents, r.cloudFormationCostComponent("Durations above 30s", "seconds", "Resource-Processing-Time", monthlyDurationSecs))

	return costComponents
}

func (r *CloudFormationStack) cloudFormationCostComponent(name, unit, usagetype string, monthlyQuantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            name,
		Unit:            unit,
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AWSCloudFormation"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s$/i", usagetype))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *CloudFormationStack) checkAWS() bool {
	return strings.Contains(strings.ToLower(r.TemplateBody), "aws::")
}

func (r *CloudFormationStack) checkAlexa() bool {
	return strings.Contains(strings.ToLower(r.TemplateBody), "alexa::")
}

func (r *CloudFormationStack) checkCustom() bool {
	return strings.Contains(strings.ToLower(r.TemplateBody), "custom::")
}
