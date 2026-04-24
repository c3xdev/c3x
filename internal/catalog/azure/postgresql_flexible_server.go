package azure

import (
	"fmt"
	"regexp"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// PostgreSQLFlexibleServer struct represents Azure PostgreSQL Flexible Server resource.
//
// Resource information: https://docs.microsoft.com/en-gb/azure/postgresql/flexible-server/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/postgresql/flexible-server/
type PostgreSQLFlexibleServer struct {
	Address string
	Region  string

	SKU              string
	Tier             string
	InstanceType     string
	InstanceVersion  string
	Storage          int64
	HighAvailability bool

	AdditionalBackupStorageGB *float64 `c3x_usage:"additional_backup_storage_gb"`
}

// CoreType returns the name of this resource type
func (r *PostgreSQLFlexibleServer) CoreType() string {
	return "PostgreSQLFlexibleServer"
}

// UsageSchema defines a list which represents the usage schema of PostgreSQLFlexibleServer.
func (r *PostgreSQLFlexibleServer) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "additional_backup_storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the PostgreSQLFlexibleServer.
// It uses the `c3x_usage` struct tags to populate data into the PostgreSQLFlexibleServer.
func (r *PostgreSQLFlexibleServer) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid PostgreSQLFlexibleServer struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *PostgreSQLFlexibleServer) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.computeCostComponent(),
		r.storageCostComponent(),
		r.backupCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// computeCostComponent returns a cost component for server compute requirements.
func (r *PostgreSQLFlexibleServer) computeCostComponent() *engine.LineItem {
	attrs := getFlexibleServerFilterAttributes(r.Tier, r.InstanceType, r.InstanceVersion)

	// Double the quantity if high availability is enabled
	quantity := decimal.NewFromInt(1)
	if r.HighAvailability {
		quantity = quantity.Mul(decimal.NewFromInt(2))
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("Compute (%s)", r.SKU),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(quantity),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Database for PostgreSQL"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(fmt.Sprintf("/^%s %s (?:-\\s)?%s/i", attrs.ProductName, attrs.TierName, attrs.Series))},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", attrs.SKUName))},
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", attrs.MeterName))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

// storageCostComponent returns a cost component for server's storage.
func (r *PostgreSQLFlexibleServer) storageCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.Storage > 0 {
		// Storage is in MB
		quantity = decimalPtr(decimal.NewFromInt(r.Storage / 1024))
	}

	return &engine.LineItem{
		Name:            "Storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Database for PostgreSQL"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Az DB for PostgreSQL Flexible Server Storage")},
				{Key: "meterName", Value: strPtr("Storage Data Stored")},
			},
		},
	}
}

// backupCostComponent returns a cost component for additional backup storage.
func (r *PostgreSQLFlexibleServer) backupCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.AdditionalBackupStorageGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.AdditionalBackupStorageGB))
	}

	return &engine.LineItem{
		Name:            "Additional backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Database for PostgreSQL"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure Database for PostgreSQL Flexible Server Backup Storage")},
				{Key: "meterName", Value: strPtr("Backup Storage LRS Data Stored")},
			},
		},
		UsageBased: true,
	}
}

// flexibleServerFilterAttributes defines CPAPI filter attributes for compute
// cost component derived from IaC provider's SKU.
type flexibleServerFilterAttributes struct {
	ProductName string
	SKUName     string
	TierName    string
	MeterName   string
	Series      string
}

// getFlexibleServerFilterAttributes returns a struct with CPAPI filter
// attributes based on values extracted from IaC provider's SKU.
func getFlexibleServerFilterAttributes(tier, instanceType, instanceVersion string) flexibleServerFilterAttributes {
	var skuName, meterName, series string

	tierName := map[string]string{
		"b":  "Burstable",
		"gp": "General Purpose",
		"mo": "Memory Optimized",
	}[tier]

	productName := "Azure Database for PostgreSQL Flexible Server"

	if tier == "b" {
		meterName = fmt.Sprintf("%s[ vcore]*", instanceType)
		skuName = instanceType
		series = "BS"
	} else {
		meterName = "vCore"

		coreRegex := regexp.MustCompile(`(\d+)`)
		match := coreRegex.FindStringSubmatch(instanceType)
		cores := match[1]
		skuName = fmt.Sprintf("%s vCore", cores)

		series = coreRegex.ReplaceAllString(instanceType, "") + instanceVersion

		if series == "Esv3" {
			productName = "Az DB for PGSQL Flexible Server"
		}
	}

	return flexibleServerFilterAttributes{
		ProductName: productName,
		SKUName:     skuName,
		TierName:    tierName,
		MeterName:   meterName,
		Series:      series,
	}
}
