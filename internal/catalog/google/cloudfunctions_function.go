package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type CloudFunctionsFunction struct {
	Address                    string
	Region                     string
	AvailableMemoryMB          *int64
	RequestDurationMs          *int64   `c3x_usage:"request_duration_ms"`
	MonthlyFunctionInvocations *int64   `c3x_usage:"monthly_function_invocations"`
	MonthlyOutboundDataGB      *float64 `c3x_usage:"monthly_outbound_data_gb"`
}

func (r *CloudFunctionsFunction) CoreType() string {
	return "CloudFunctionsFunction"
}

func (r *CloudFunctionsFunction) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "request_duration_ms", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_function_invocations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_outbound_data_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *CloudFunctionsFunction) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudFunctionsFunction) BuildResource() *engine.Estimate {
	memorySize := decimal.NewFromInt(256)
	if r.AvailableMemoryMB != nil {
		memorySize = decimal.NewFromInt(*r.AvailableMemoryMB)
	}

	var cpuMapping = map[int]decimal.Decimal{
		128:  decimal.NewFromInt(200),
		256:  decimal.NewFromInt(400),
		512:  decimal.NewFromInt(800),
		1024: decimal.NewFromInt(1400),
		2048: decimal.NewFromInt(2400),
		4096: decimal.NewFromInt(4800),
	}

	cpuSize := cpuMapping[int(memorySize.IntPart())]

	requestDuration := decimal.NewFromInt(100)
	if r.RequestDurationMs != nil {

		requestDuration = decimal.NewFromInt(*r.RequestDurationMs).Div(decimal.NewFromInt(100)).Ceil().Mul(decimal.NewFromFloat(100))
	}

	var invocations, monthlyCPUUsage, monthlyMemoryUsage *decimal.Decimal
	if r.MonthlyFunctionInvocations != nil {
		invocations = decimalPtr(decimal.NewFromInt(*r.MonthlyFunctionInvocations))
		monthlyCPUUsage = decimalPtr(r.calculateGHzSeconds(cpuSize, requestDuration, *invocations))
		monthlyMemoryUsage = decimalPtr(r.calculateGBSeconds(memorySize, requestDuration, *invocations))
	}

	var networkEgress *decimal.Decimal
	if r.MonthlyOutboundDataGB != nil {
		networkEgress = decimalPtr(decimal.NewFromFloat(*r.MonthlyOutboundDataGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "CPU",
				Unit:            "GHz-seconds",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyCPUUsage,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Cloud Run Functions"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: regexPtr("\\(1st Gen\\) CPU")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Memory",
				Unit:            "GB-seconds",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyMemoryUsage,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Cloud Run Functions"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: regexPtr("\\(1st Gen\\) Memory")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Invocations",
				Unit:            "invocations",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: invocations,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud Run Functions"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: regexPtr("\\(1st Gen\\) Invocations")},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("2000000"),
				},
				UsageBased: true,
			},
			{
				Name:            "Outbound data transfer",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: networkEgress,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Cloud Run Functions"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: regexPtr("\\(1st Gen\\) Network Data Transfer Out")},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("5"),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *CloudFunctionsFunction) calculateGBSeconds(memorySize decimal.Decimal, averageRequestDuration decimal.Decimal, monthlyRequests decimal.Decimal) decimal.Decimal {
	gb := memorySize.Div(decimal.NewFromInt(1024))
	seconds := averageRequestDuration.Div(decimal.NewFromInt(1000))
	return monthlyRequests.Mul(gb).Mul(seconds)
}

func (r *CloudFunctionsFunction) calculateGHzSeconds(memorySize decimal.Decimal, averageRequestDuration decimal.Decimal, monthlyRequests decimal.Decimal) decimal.Decimal {
	gb := memorySize.Div(decimal.NewFromInt(1000))
	seconds := averageRequestDuration.Div(decimal.NewFromInt(1000))
	return monthlyRequests.Mul(gb).Mul(seconds)
}
