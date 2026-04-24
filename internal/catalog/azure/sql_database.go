package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

const (
	sqlServerlessTier     = "general purpose - serverless"
	sqlHyperscaleTier     = "hyperscale"
	sqlGeneralPurposeTier = "general purpose"
)

var (
	mssqlTierMapping = map[string]string{
		"b": "Basic",
		"p": "Premium",
		"s": "Standard",
	}

	mssqlPremiumDTUIncludedStorage = map[string]float64{
		"p1":  500,
		"p2":  500,
		"p4":  500,
		"p6":  500,
		"p11": 4096,
		"p15": 4096,
	}

	mssqlStorageRedundancyTypeMapping = map[string]string{
		"geo":   "RA-GRS",
		"local": "LRS",
		"zone":  "ZRS",
	}
)

// SQLDatabase represents an Azure SQL database instance.
//
// More resource information here: https://azure.microsoft.com/en-gb/products/azure-sql/database/
// Pricing information here: https://azure.microsoft.com/en-gb/pricing/details/azure-sql-database/single/
type SQLDatabase struct {
	Address           string
	Region            string
	SKU               string
	IsElasticPool     bool
	LicenseType       string
	Tier              string
	Family            string
	Cores             *int64
	MaxSizeGB         *float64
	ReadReplicaCount  *int64
	ZoneRedundant     bool
	BackupStorageType string
	IsDevTest         bool

	// ExtraDataStorageGB represents a usage cost of additional backup storage used by the sql database.
	ExtraDataStorageGB *float64 `c3x_usage:"extra_data_storage_gb"`
	// MonthlyVCoreHours represents a usage param that allows users to define how many hours of usage a serverless sql database instance uses.
	MonthlyVCoreHours *int64 `c3x_usage:"monthly_vcore_hours"`
	// LongTermRetentionStorageGB defines a usage param that allows users to define how many GB of cold storage the database uses.
	// This is storage that can be kept for up to 10 years.
	LongTermRetentionStorageGB *int64 `c3x_usage:"long_term_retention_storage_gb"`
	// BackupStorageGB defines a usage param that allows users to define how many GB Point-In-Time Restore (PITR) backup storage the database uses.
	BackupStorageGB *int64 `c3x_usage:"backup_storage_gb"`
}

// PopulateUsage parses the u engine.ConsumptionProfile into the SQLDatabase.
func (r *SQLDatabase) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SQLDatabase) CoreType() string {
	return "SQLDatabase"
}

func (r *SQLDatabase) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "extra_data_storage_gb", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_vcore_hours", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "long_term_retention_storage_gb", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "backup_storage_gb", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// BuildResource builds a engine.Estimate from a valid SQLDatabase.
// It returns a SQLDatabase as a *engine.Estimate with cost components initialized.
//
// SQLDatabase splits pricing into two different models. DTU & vCores.
//
//	Database Transaction Unit (DTU) is made a performance metric representing a mixture of performance metrics
//	in Azure SQL. Some include: CPU, I/O, Memory. DTU is used as Azure tries to simplify billing by using a single metric.
//
//	Virtual Core (vCore) pricing is designed to translate from on premise hardware metrics (cores) into the cloud
//	SQL instance. vCore is designed to allow users to better estimate their resource limits, e.g. RAM.
//
// SQL databases that follow a DTU pricing model have the following costs associated with them:
//
//  1. Costs based on the number of DTUs that the sql database has
//  2. Extra backup data costs - this is configured using SQLDatabase.ExtraDataStorageGB
//  3. Long term data backup costs - this is configured using SQLDatabase.LongTermRetentionStorageGB
//
// SQL databases that follow a vCore pricing model have the following costs associated with them:
//
//  1. Costs based on the number of vCores the resource has
//  2. Extra pricing if any database read replicas have been provisioned
//  3. Additional charge for SQL Server licensing based on vCores amount
//  4. Charges for storage used
//  5. Charges for long term data backup - this is configured using SQLDatabase.LongTermRetentionStorageGB
//
// This method is called after the resource is initialized by an IaC provider. SQLDatabase is used by both mssql_database
// and sql_database Terraform catalog.
func (r *SQLDatabase) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: r.costComponents(),
	}
}

func (r *SQLDatabase) costComponents() []*engine.LineItem {
	if r.IsElasticPool {
		return r.elasticPoolCostComponents()
	}

	if r.Cores != nil {
		return r.vCoreCostComponents()
	}

	return r.dtuCostComponents()
}

func (r *SQLDatabase) dtuCostComponents() []*engine.LineItem {
	skuName := strings.ToLower(r.SKU)
	if skuName == "basic" {
		skuName = "b"
	}

	// This is a bit of a hack, but the Azure pricing API returns the price per day
	// and the Azure pricing calculator uses 730 hours to show the cost
	// so we need to convert the price per day to price per hour.
	// Use precision 24 to avoid rounding errors later since the default decimal precision is 16.
	daysInMonth := engine.HourToMonthUnitMultiplier.DivRound(decimal.NewFromInt(24), 24)

	name := fmt.Sprintf("Compute (%s)", strings.ToTitle(r.SKU))
	purchaseOption := priceFilterConsumption
	if r.IsDevTest {
		name = fmt.Sprintf("Compute (dev/test, %s)", strings.ToTitle(r.SKU))
		purchaseOption = priceFilterDevTestConsumption
	}

	costComponents := []*engine.LineItem{
		{
			Name:            name,
			Unit:            "hours",
			UnitMultiplier:  daysInMonth.DivRound(engine.HourToMonthUnitMultiplier, 24),
			MonthlyQuantity: decimalPtr(daysInMonth),
			ProductFilter: r.productFilter([]*engine.AttributeMatch{
				{Key: "productName", ValueRegex: regexPtr("^SQL Database Single")},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", skuName))},
				{Key: "meterName", ValueRegex: regexPtr("DTU(s)?$")},
			}),
			PriceFilter: purchaseOption,
		},
	}

	var extraStorageGB float64

	if !strings.HasPrefix(skuName, "b") && r.ExtraDataStorageGB != nil {
		extraStorageGB = *r.ExtraDataStorageGB
	} else if strings.HasPrefix(skuName, "s") && r.MaxSizeGB != nil {
		includedStorageGB := 250.0
		extraStorageGB = *r.MaxSizeGB - includedStorageGB
	} else if strings.HasPrefix(skuName, "p") && r.MaxSizeGB != nil {
		includedStorageGB, ok := mssqlPremiumDTUIncludedStorage[skuName]
		if ok {
			extraStorageGB = *r.MaxSizeGB - includedStorageGB
		}
	}

	if extraStorageGB > 0 {
		c := r.extraDataStorageCostComponent(extraStorageGB)
		if c != nil {
			costComponents = append(costComponents, c)
		}
	}

	costComponents = append(costComponents, r.longTermRetentionCostComponent())
	costComponents = append(costComponents, r.pitrBackupCostComponent())

	return costComponents
}

func (r *SQLDatabase) vCoreCostComponents() []*engine.LineItem {
	costComponents := r.computeHoursCostComponents()

	if strings.ToLower(r.Tier) == sqlHyperscaleTier {
		costComponents = append(costComponents, r.readReplicaCostComponent())
	}

	if strings.ToLower(r.Tier) != sqlServerlessTier && strings.ToLower(r.LicenseType) == "licenseincluded" {
		costComponents = append(costComponents, r.sqlLicenseCostComponent())
	}

	costComponents = append(costComponents, r.storageCostComponent())

	if strings.ToLower(r.Tier) != sqlHyperscaleTier {
		costComponents = append(costComponents, r.longTermRetentionCostComponent())
		costComponents = append(costComponents, r.pitrBackupCostComponent())
	}

	return costComponents
}

func (r *SQLDatabase) elasticPoolCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		r.longTermRetentionCostComponent(),
		r.pitrBackupCostComponent(),
	}
}

func (r *SQLDatabase) computeHoursCostComponents() []*engine.LineItem {
	if strings.ToLower(r.Tier) == sqlServerlessTier {
		return r.serverlessComputeHoursCostComponents()
	}

	return r.provisionedComputeCostComponents()
}

func (r *SQLDatabase) serverlessComputeHoursCostComponents() []*engine.LineItem {
	productNameRegex := fmt.Sprintf("/%s - %s/", r.Tier, r.Family)

	var vCoreHours *decimal.Decimal
	if r.MonthlyVCoreHours != nil {
		vCoreHours = decimalPtr(decimal.NewFromInt(*r.MonthlyVCoreHours))
	}

	name := fmt.Sprintf("Compute (serverless, %s)", r.SKU)
	purchaseOption := priceFilterConsumption
	if r.IsDevTest && strings.ToLower(r.LicenseType) != "licenseincluded" {
		name = fmt.Sprintf("Compute (dev/test, serverless, %s)", r.SKU)
		purchaseOption = priceFilterDevTestConsumption
	}

	costComponents := []*engine.LineItem{
		{
			Name:            name,
			Unit:            "vCore-hours",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: vCoreHours,
			ProductFilter: r.productFilter([]*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(productNameRegex)},
				{Key: "skuName", Value: strPtr("1 vCore")},
				{Key: "meterName", ValueRegex: regexPtr("^(?!.* - Free$).*$")},
			}),
			PriceFilter: purchaseOption,
			UsageBased:  true,
		},
	}

	// Zone redundancy is free for premium and business critical tiers
	if r.ZoneRedundant {
		costComponents = append(costComponents, &engine.LineItem{
			Name:            fmt.Sprintf("Zone redundancy (serverless, %s)", r.SKU),
			Unit:            "vCore-hours",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: vCoreHours,
			ProductFilter: r.productFilter([]*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(productNameRegex)},
				{Key: "skuName", Value: strPtr("1 vCore Zone Redundancy")},
				{Key: "meterName", ValueRegex: regexPtr("^(?!.* - Free$).*$")},
			}),
			PriceFilter: priceFilterConsumption,
		})
	}

	return costComponents
}

func (r *SQLDatabase) provisionedComputeCostComponents() []*engine.LineItem {
	var cores int64
	if r.Cores != nil {
		cores = *r.Cores
	}

	productNameRegex := fmt.Sprintf("/%s - %s/", r.Tier, r.Family)
	name := fmt.Sprintf("Compute (provisioned, %s)", r.SKU)
	purchaseOption := priceFilterConsumption
	if r.IsDevTest && strings.ToLower(r.LicenseType) != "licenseincluded" {
		name = fmt.Sprintf("Compute (dev/test, provisioned, %s)", r.SKU)
		purchaseOption = priceFilterDevTestConsumption
	}

	costComponents := []*engine.LineItem{
		{
			Name:           name,
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: r.productFilter([]*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(productNameRegex)},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("%d vCore", cores))},
			}),
			PriceFilter: purchaseOption,
		},
	}

	// Zone redundancy is free for premium and business critical tiers
	if strings.EqualFold(r.Tier, sqlGeneralPurposeTier) && r.ZoneRedundant {
		costComponents = append(costComponents, &engine.LineItem{
			Name:           fmt.Sprintf("Zone redundancy (provisioned, %s)", r.SKU),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: r.productFilter([]*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(productNameRegex)},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("%d vCore Zone Redundancy", cores))},
			}),
			PriceFilter: priceFilterConsumption,
		})
	}

	return costComponents
}

func (r *SQLDatabase) readReplicaCostComponent() *engine.LineItem {
	productNameRegex := fmt.Sprintf("/%s - %s/", r.Tier, r.Family)
	skuName := mssqlSkuName(*r.Cores, r.ZoneRedundant)

	var replicaCount *decimal.Decimal
	if r.ReadReplicaCount != nil {
		replicaCount = decimalPtr(decimal.NewFromInt(*r.ReadReplicaCount))
	}

	return &engine.LineItem{
		Name:           "Read replicas",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: replicaCount,
		ProductFilter: r.productFilter([]*engine.AttributeMatch{
			{Key: "productName", ValueRegex: strPtr(productNameRegex)},
			{Key: "skuName", Value: strPtr(skuName)},
		}),
		PriceFilter: priceFilterConsumption,
	}
}

func (r *SQLDatabase) longTermRetentionCostComponent() *engine.LineItem {
	var retention *decimal.Decimal
	if r.LongTermRetentionStorageGB != nil {
		retention = decimalPtr(decimal.NewFromInt(*r.LongTermRetentionStorageGB))
	}

	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(r.BackupStorageType)]
	if !ok {
		logging.Logger.Warn().Msgf("Unrecognized backup storage type '%s'", r.BackupStorageType)
		redundancyType = "RA-GRS"
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("Long-term retention (%s)", redundancyType),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: retention,
		ProductFilter: r.productFilter([]*engine.AttributeMatch{
			{Key: "productName", Value: strPtr("SQL Database - LTR Backup Storage")},
			{Key: "skuName", Value: strPtr(fmt.Sprintf("Backup %s", redundancyType))},
			{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s Data Stored", redundancyType))},
		}),
		PriceFilter: priceFilterConsumption,
		UsageBased:  true,
	}
}

func (r *SQLDatabase) pitrBackupCostComponent() *engine.LineItem {
	var pitrGB *decimal.Decimal
	if r.BackupStorageGB != nil {
		pitrGB = decimalPtr(decimal.NewFromInt(*r.BackupStorageGB))
	}

	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(r.BackupStorageType)]
	if !ok {
		logging.Logger.Warn().Msgf("Unrecognized backup storage type '%s'", r.BackupStorageType)
		redundancyType = "RA-GRS"
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("PITR backup storage (%s)", redundancyType),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: pitrGB,
		ProductFilter: r.productFilter([]*engine.AttributeMatch{
			{Key: "productName", ValueRegex: regexPtr("PITR Backup Storage")},
			{Key: "skuName", Value: strPtr(fmt.Sprintf("Backup %s", redundancyType))},
			{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s Data Stored", redundancyType))},
		}),
		PriceFilter: priceFilterConsumption,
		UsageBased:  true,
	}
}

func (r *SQLDatabase) extraDataStorageCostComponent(extraStorageGB float64) *engine.LineItem {
	tier := r.Tier
	if tier == "" {
		var ok bool
		tier, ok = mssqlTierMapping[strings.ToLower(r.SKU)[:1]]

		if !ok {
			logging.Logger.Warn().Msgf("Unrecognized tier for SKU '%s' for resource %s", r.SKU, r.Address)
			return nil
		}
	}

	return mssqlExtraDataStorageCostComponent(r.Region, tier, extraStorageGB)
}

func (r *SQLDatabase) sqlLicenseCostComponent() *engine.LineItem {
	return mssqlLicenseCostComponent(r.Region, r.Cores, r.Tier)
}

func (r *SQLDatabase) storageCostComponent() *engine.LineItem {
	return mssqlStorageCostComponent(r.Region, r.Tier, r.ZoneRedundant, r.MaxSizeGB)
}

func (r *SQLDatabase) productFilter(filters []*engine.AttributeMatch) *engine.ProductSelector {
	return mssqlProductFilter(r.Region, filters)
}

func mssqlSkuName(cores int64, zoneRedundant bool) string {
	sku := fmt.Sprintf("%d vCore", cores)

	if zoneRedundant {
		sku += " Zone Redundancy"
	}
	return sku
}

func mssqlProductFilter(region string, filters []*engine.AttributeMatch) *engine.ProductSelector {
	return &engine.ProductSelector{
		VendorName:       strPtr(vendorName),
		Region:           strPtr(region),
		Service:          strPtr("SQL Database"),
		ProductFamily:    strPtr("Databases"),
		AttributeFilters: filters,
	}
}

func mssqlExtraDataStorageCostComponent(region string, tier string, extraStorageGB float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Extra data storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromFloat(extraStorageGB)),
		ProductFilter: mssqlProductFilter(region, []*engine.AttributeMatch{
			{Key: "productName", ValueRegex: strPtr(fmt.Sprintf("/SQL Database %s - Storage/i", tier))},
			{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", tier))},
			{Key: "meterName", Value: strPtr("Data Stored")},
		}),
		PriceFilter: priceFilterConsumption,
		UsageBased:  true,
	}
}

func mssqlLicenseCostComponent(region string, cores *int64, tier string) *engine.LineItem {
	licenseRegion := "Global"
	if strings.Contains(region, "usgov") {
		licenseRegion = "US Gov"
	}

	if strings.Contains(region, "china") {
		licenseRegion = "China"
	}

	if strings.Contains(region, "germany") {
		licenseRegion = "Germany"
	}

	coresVal := int64(1)
	if cores != nil {
		coresVal = *cores
	}

	return &engine.LineItem{
		Name:           "SQL license",
		Unit:           "vCore-hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(coresVal)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(licenseRegion),
			Service:       strPtr("SQL Database"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", ValueRegex: strPtr(fmt.Sprintf("/%s - %s/", tier, "SQL License"))},
			},
		},
		PriceFilter: priceFilterConsumption,
	}
}

func mssqlStorageCostComponent(region string, tier string, zoneRedundant bool, maxSizeGB *float64) *engine.LineItem {
	storageGB := decimalPtr(decimal.NewFromInt(5))
	if maxSizeGB != nil {
		storageGB = decimalPtr(decimal.NewFromFloat(*maxSizeGB))
	}

	storageTier := tier
	if strings.EqualFold(tier, sqlServerlessTier) {
		storageTier = "General Purpose"
	}

	skuName := storageTier
	if (strings.EqualFold(tier, sqlGeneralPurposeTier) || strings.EqualFold(tier, sqlServerlessTier)) && zoneRedundant {
		skuName += " Zone Redundancy"
	}

	productNameRegex := fmt.Sprintf("/%s - Storage/", storageTier)

	filters := []*engine.AttributeMatch{
		{Key: "productName", ValueRegex: strPtr(productNameRegex)},
		{Key: "skuName", Value: strPtr(skuName)},
		{Key: "meterName", ValueRegex: regexPtr("Data Stored$")},
	}

	if skuName == "Hyperscale" {
		filters = append(filters, &engine.AttributeMatch{Key: "armSkuName", Value: strPtr(skuName)})
	}

	return &engine.LineItem{
		Name:            "Storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageGB,
		ProductFilter:   mssqlProductFilter(region, filters),
	}
}
