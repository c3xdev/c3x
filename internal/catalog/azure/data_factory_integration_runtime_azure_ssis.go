package azure

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// DataFactoryIntegrationRuntimeAzureSSIS struct represents Data Factory's
// Azure-SSIS runtime.
//
// Resource information: https://azure.microsoft.com/en-us/services/data-factory/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/data-factory/ssis/
type DataFactoryIntegrationRuntimeAzureSSIS struct {
	Address string
	Region  string

	Instances       int64
	InstanceType    string
	Enterprise      bool
	LicenseIncluded bool
}

func (r *DataFactoryIntegrationRuntimeAzureSSIS) CoreType() string {
	return "DataFactoryIntegrationRuntimeAzureSSIS"
}

func (r *DataFactoryIntegrationRuntimeAzureSSIS) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the DataFactoryIntegrationRuntimeAzureSSIS.
// It uses the `c3x_usage` struct tags to populate data into the DataFactoryIntegrationRuntimeAzureSSIS.
func (r *DataFactoryIntegrationRuntimeAzureSSIS) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid DataFactoryIntegrationRuntimeAzureSSIS struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *DataFactoryIntegrationRuntimeAzureSSIS) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.computeCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// computeCostComponent returns a cost component for cluster configuration.
func (r *DataFactoryIntegrationRuntimeAzureSSIS) computeCostComponent() *engine.LineItem {
	tier := "Standard"
	if r.Enterprise {
		tier = "Enterprise"
	}

	license := "License Included"
	licenseTitle := ", license included"
	if !r.LicenseIncluded {
		license = "AHB"
		licenseTitle = ""
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("Compute (%s, %s%s)", r.InstanceType, tier, licenseTitle),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.Instances)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(license)},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", r.InstanceType))},
				{Key: "productName", ValueRegex: regexPtr(fmt.Sprintf("^SSIS %s", tier))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
