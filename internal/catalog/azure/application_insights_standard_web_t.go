package azure

import (
	"fmt"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// ApplicationInsightsStandardWebTest struct represents an Application Insights Standard WebTest.
//
// Resource information: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_insights_standard_web_test
// Pricing information: https://azure.microsoft.com/en-in/pricing/details/monitor/
type ApplicationInsightsStandardWebTest struct {
	Address string
	Region  string

	Enabled   bool
	Frequency int64
}

// CoreType returns the name of this resource type
func (r *ApplicationInsightsStandardWebTest) CoreType() string {
	return "ApplicationInsightsStandardWebTest"
}

// UsageSchema defines a list which represents the usage schema of ApplicationInsightsStandardWebTest.
func (r *ApplicationInsightsStandardWebTest) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ApplicationInsightsStandardWebTest.
// It uses the `c3x_usage` struct tags to populate data into the ApplicationInsightsStandardWebTest.
func (r *ApplicationInsightsStandardWebTest) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ApplicationInsightsStandardWebTest struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ApplicationInsightsStandardWebTest) BuildResource() *engine.Estimate {
	var costComponents []*engine.LineItem

	if r.Enabled {
		secondsPerMonth := int64(730 * 60 * 60) // 730 hours * 60 minutes * 60 seconds
		tests := secondsPerMonth / r.Frequency

		costComponents = append(costComponents, &engine.LineItem{
			Name:            fmt.Sprintf("Standard web test (%d second frequency)", r.Frequency),
			Unit:            "tests",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(tests)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Azure Monitor"),
				ProductFamily: strPtr("Management and Governance"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", Value: strPtr("Standard Web Test")},
				},
			},
		})
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
