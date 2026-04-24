package azure

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ApplicationInsightsWebTest struct {
	Address string
	Region  string
	Kind    string
	Enabled bool
}

func (r *ApplicationInsightsWebTest) CoreType() string {
	return "ApplicationInsightsWebTest"
}

func (r *ApplicationInsightsWebTest) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ApplicationInsightsWebTest) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ApplicationInsightsWebTest) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	if r.Kind != "" {
		if strings.ToLower(r.Kind) == "multistep" && r.Enabled {
			costComponents = append(costComponents, &engine.LineItem{
				Name:            "Multi-step web test",
				Unit:            "test",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Application Insights"),
					ProductFamily: strPtr("Management and Governance"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Multi-step Web Test"))},
						{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Enterprise"))},
					},
				},
			})
		}
	}

	if len(costComponents) == 0 {
		return &engine.Estimate{
			Name:        r.Address,
			IsSkipped:   true,
			NoPrice:     true,
			UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}

}
