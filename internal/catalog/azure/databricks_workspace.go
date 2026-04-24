package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type DatabricksWorkspace struct {
	Address                        string
	Region                         string
	SKU                            string
	MonthlyAllPurposeComputeDBUHrs *int64 `c3x_usage:"monthly_all_purpose_compute_dbu_hrs"`
	MonthlyJobsComputeDBUHrs       *int64 `c3x_usage:"monthly_jobs_compute_dbu_hrs"`
	MonthlyJobsLightComputeDBUHrs  *int64 `c3x_usage:"monthly_jobs_light_compute_dbu_hrs"`
}

func (r *DatabricksWorkspace) CoreType() string {
	return "DatabricksWorkspace"
}

func (r *DatabricksWorkspace) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_all_purpose_compute_dbu_hrs", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_jobs_compute_dbu_hrs", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_jobs_light_compute_dbu_hrs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *DatabricksWorkspace) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DatabricksWorkspace) BuildResource() *engine.Estimate {
	var costComponents []*engine.LineItem

	sku := cases.Title(language.English).String(r.SKU)

	if sku == "Trial" {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	var allPurpose, jobs, jobsLight *decimal.Decimal

	if r.MonthlyAllPurposeComputeDBUHrs != nil {
		allPurpose = decimalPtr(decimal.NewFromInt(*r.MonthlyAllPurposeComputeDBUHrs))
	}
	costComponents = append(costComponents, r.databricksCostComponent(
		"All-purpose compute DBUs",
		fmt.Sprintf("%s All-purpose Compute", sku),
		allPurpose,
	))

	if r.MonthlyJobsComputeDBUHrs != nil {
		jobs = decimalPtr(decimal.NewFromInt(*r.MonthlyJobsComputeDBUHrs))
	}
	costComponents = append(costComponents, r.databricksCostComponent(
		"Jobs compute DBUs",
		fmt.Sprintf("%s Jobs Compute", sku),
		jobs,
	))

	if r.MonthlyJobsLightComputeDBUHrs != nil {
		jobsLight = decimalPtr(decimal.NewFromInt(*r.MonthlyJobsLightComputeDBUHrs))
	}
	costComponents = append(costComponents, r.databricksCostComponent(
		"Jobs light compute DBUs",
		fmt.Sprintf("%s Jobs Light Compute", sku),
		jobsLight,
	))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *DatabricksWorkspace) databricksCostComponent(name, skuName string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "DBU-hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Databricks"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuName)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}
