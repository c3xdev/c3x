package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

type GrafanaWorkspace struct {
	Address                      string
	Region                       string
	License                      string
	EditorsAdministratorLicenses *int64 `c3x_usage:"editors_administrator_licenses"`
	ViewerLicenses               *int64 `c3x_usage:"viewer_licenses"`
}

func (r *GrafanaWorkspace) CoreType() string {
	return "GrafanaWorkspace"
}

func (r *GrafanaWorkspace) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "editors_administrator_licenses", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "viewer_licenses", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *GrafanaWorkspace) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *GrafanaWorkspace) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	var editorLicenses *decimal.Decimal
	if r.EditorsAdministratorLicenses != nil && *r.EditorsAdministratorLicenses > 0 {
		editorLicenses = decimalPtr(decimal.NewFromInt(*r.EditorsAdministratorLicenses))
	} else if r.EditorsAdministratorLicenses == nil {
		editorLicenses = decimalPtr(decimal.NewFromInt(1))
	}

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Editor/administrator licenses",
		Unit:            "users",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: editorLicenses,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonGrafana"),
			ProductFamily: strPtr("User Fees"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("Grafana:EditorUser$")},
			},
		},
	})

	var viewerLicenses *decimal.Decimal
	if r.ViewerLicenses != nil && *r.ViewerLicenses > 0 {
		viewerLicenses = decimalPtr(decimal.NewFromInt(*r.ViewerLicenses))
	}

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Viewer licenses",
		Unit:            "users",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: viewerLicenses,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonGrafana"),
			ProductFamily: strPtr("User Fees"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("Grafana:ViewerUser$")},
			},
		},
		UsageBased: true,
	})

	if strings.EqualFold(r.License, "ENTERPRISE") {
		var enterprisePluginsQty decimal.Decimal
		if editorLicenses != nil {
			enterprisePluginsQty = enterprisePluginsQty.Add(*editorLicenses)
		}
		if viewerLicenses != nil {
			enterprisePluginsQty = enterprisePluginsQty.Add(*viewerLicenses)
		}

		if enterprisePluginsQty.GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, &engine.LineItem{
				Name:            "Enterprise plugins licenses",
				Unit:            "users",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: &enterprisePluginsQty,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonGrafana"),
					ProductFamily: strPtr("User Fees"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: regexPtr("Grafana:EnterprisePluginsUser$")},
					},
				},
			})
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
