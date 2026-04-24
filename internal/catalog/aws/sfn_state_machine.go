package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"

	"strings"
)

type SFnStateMachine struct {
	Address            string
	Region             string
	Type               string
	MonthlyRequests    *int64 `c3x_usage:"monthly_requests"`
	WorkflowDurationMs *int64 `c3x_usage:"workflow_duration_ms"`
	MemoryMB           *int64 `c3x_usage:"memory_mb"`
	MonthlyTransitions *int64 `c3x_usage:"monthly_transitions"`
}

func (r *SFnStateMachine) CoreType() string {
	return "SFnStateMachine"
}

func (r *SFnStateMachine) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "workflow_duration_ms", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "memory_mb", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_transitions", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SFnStateMachine) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SFnStateMachine) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	tier := r.Type
	if tier == "" {
		tier = "STANDARD"
	}

	if strings.ToLower(tier) == "standard" {
		var transitions *decimal.Decimal
		if r.MonthlyTransitions != nil {
			transitions = decimalPtr(decimal.NewFromInt(*r.MonthlyTransitions))
		}
		costComponents = append(costComponents, r.transistionsCostComponent(transitions))
	}

	if strings.ToLower(tier) == "express" {
		var requests *decimal.Decimal
		if r.MonthlyRequests != nil {
			requests = decimalPtr(decimal.NewFromInt(*r.MonthlyRequests))
		}
		costComponents = append(costComponents, r.requestsCostComponent(requests))

		if r.WorkflowDurationMs != nil && r.MonthlyRequests != nil && r.MemoryMB != nil {

			memoryRequest := decimalPtr(decimal.NewFromInt(*r.MemoryMB))
			duration := decimalPtr(decimal.NewFromInt(*r.WorkflowDurationMs))
			gbSeconds := decimalPtr(r.calculateGBSeconds(*memoryRequest, *duration, *requests))

			pushLimits := []int{3600000, 14400000}
			pushQuantities := usage.CalculateTierBuckets(*gbSeconds, pushLimits)

			costComponents = append(costComponents, r.durationCostComponent("Duration (first 1K)", "0", &pushQuantities[0]))
			if pushQuantities[1].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, r.durationCostComponent("Duration (next 4K)", "3600000", &pushQuantities[1]))
			}
			if pushQuantities[2].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, r.durationCostComponent("Duration (over 5K)", "18000000", &pushQuantities[2]))
			}
		} else {
			var unknown *decimal.Decimal
			costComponents = append(costComponents, r.durationCostComponent("Duration (first 1K)", "0", unknown))
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *SFnStateMachine) transistionsCostComponent(quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Transitions",
		Unit:            "1K transitions",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonStates"),
			ProductFamily: strPtr("AWS Step Functions"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/StateTransition/")},
			},
		},
		UsageBased: true,
	}
}

func (r *SFnStateMachine) requestsCostComponent(quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Requests",
		Unit:            "1M requests",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonStates"),
			ProductFamily: strPtr("AWS Step Functions"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/StepFunctions-Request/")},
			},
		},
		UsageBased: true,
	}
}

func (r *SFnStateMachine) durationCostComponent(name string, startUsageAmt string, gbSeconds *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB-hours",
		UnitMultiplier:  decimal.NewFromInt(3600),
		MonthlyQuantity: gbSeconds,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonStates"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/StepFunctions-GB-Second/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(startUsageAmt),
		},
		UsageBased: true,
	}
}

func (r *SFnStateMachine) calculateGBSeconds(memorySize decimal.Decimal, averageRequestDuration decimal.Decimal, monthlyRequests decimal.Decimal) decimal.Decimal {

	if memorySize.LessThan(decimal.NewFromInt(64)) {
		memorySize = decimal.NewFromInt(64)
	}
	roundedMemory := memorySize.Div(decimal.NewFromInt(64)).Ceil().Mul(decimal.NewFromInt(64))

	roundedDuration := averageRequestDuration.Div(decimal.NewFromInt(100)).Ceil().Mul(decimal.NewFromInt(100))
	durationSeconds := monthlyRequests.Mul(roundedDuration).Mul(decimal.NewFromFloat(0.001))
	gbSeconds := durationSeconds.Mul(roundedMemory).Div(decimal.NewFromInt(1024))
	return gbSeconds
}
