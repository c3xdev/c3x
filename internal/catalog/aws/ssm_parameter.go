package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type SSMParameter struct {
	Address                string
	Tier                   string
	Region                 string
	ParameterStorageHrs    *int64  `c3x_usage:"parameter_storage_hrs"`
	APIThroughputLimit     *string `c3x_usage:"api_throughput_limit"`
	MonthlyAPIInteractions *int64  `c3x_usage:"monthly_api_interactions"`
}

func (r *SSMParameter) CoreType() string {
	return "SSMParameter"
}

func (r *SSMParameter) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "parameter_storage_hrs", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "api_throughput_limit", ValueType: engine.String, DefaultValue: "standard"},
		{Key: "monthly_api_interactions", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SSMParameter) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SSMParameter) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	throughputLimit := ""

	if r.APIThroughputLimit != nil {
		throughputLimit = strings.ToLower(*r.APIThroughputLimit)

		if throughputLimit != "standard" && throughputLimit != "advanced" && throughputLimit != "higher" {
			logging.Logger.Warn().Msgf("Skipping resource %s. Unrecognized api_throughput_limit %s, expecting standard, advanced or higher", r.Address, *r.APIThroughputLimit)
			return nil
		}
	}

	if throughputLimit == "" {
		throughputLimit = r.tierValue()
	}

	if r.tierValue() != "standard" {
		costComponents = append(costComponents, r.parameterStorageCostComponent())
		costComponents = append(costComponents, r.apiThroughputCostComponent(throughputLimit))
	}

	if len(costComponents) == 0 {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *SSMParameter) tierValue() string {
	if r.Tier == "" {
		return "standard"
	}

	return strings.ToLower(r.Tier)
}

func (r *SSMParameter) parameterStorageCostComponent() *engine.LineItem {
	parameterStorageHours := decimal.NewFromInt(730)
	if r.ParameterStorageHrs != nil {
		parameterStorageHours = decimal.NewFromInt(*r.ParameterStorageHrs)
	}

	return &engine.LineItem{
		Name:            "Parameter storage (advanced)",
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: &parameterStorageHours,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSSystemsManager"),
			ProductFamily: strPtr("AWS Systems Manager"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/PS-Advanced-Param-Tier1/")},
			},
		},
		UsageBased: true,
	}
}

func (r *SSMParameter) apiThroughputCostComponent(throughputLimit string) *engine.LineItem {
	var monthlyAPIInteractions *decimal.Decimal
	if r.MonthlyAPIInteractions != nil {
		monthlyAPIInteractions = decimalPtr(decimal.NewFromInt(*r.MonthlyAPIInteractions))
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("API interactions (%s)", throughputLimit),
		Unit:            "10k interactions",
		UnitMultiplier:  decimal.NewFromInt(10000),
		MonthlyQuantity: monthlyAPIInteractions,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSSystemsManager"),
			ProductFamily: strPtr("API Request"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/PS-Param-Processed-Tier2/")},
			},
		},
		UsageBased: true,
	}
}
