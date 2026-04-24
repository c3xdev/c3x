package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

var mssqlElasticPoolPremiumDTUIncludedStorage = map[int]float64{
	125:  250,
	250:  500,
	500:  750,
	1000: 1024,
	1500: 1536,
	2000: 2048,
	2500: 2560,
	3000: 3072,
	3500: 3584,
	4000: 4096,
}

// MSSQLElasticPool represents an Azure MSSQL Elastic Pool instance.
//
// More resource information here: https://azure.microsoft.com/en-gb/products/azure-sql/database/
// Pricing information here: https://azure.microsoft.com/en-gb/pricing/details/azure-sql-database/
type MSSQLElasticPool struct {
	Address       string
	Region        string
	SKU           string
	LicenseType   string
	Tier          string
	Family        string
	Cores         *int64
	DTUCapacity   *int64
	MaxSizeGB     *float64
	ZoneRedundant bool
}

func (r *MSSQLElasticPool) CoreType() string {
	return "MSSQLElasticPool"
}

func (r *MSSQLElasticPool) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the MSSQLElasticPool.
func (r *MSSQLElasticPool) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid MSSQLElasticPool.
// It returns a MSSQLElasticPool as a *engine.Estimate with cost components initialized.
//
// MSSQLElasticPool splits pricing into two different models. DTU & vCores.
//
//	Database Transaction Unit (DTU) is made a performance metric representing a mixture of performance metrics
//	in Azure SQL. Some include: CPU, I/O, Memory. DTU is used as Azure tries to simplify billing by using a single metric.
//
//	Virtual Core (vCore) pricing is designed to translate from on premise hardware metrics (cores) into the cloud
//	SQL instance. vCore is designed to allow users to better estimate their resource limits, e.g. RAM.
//
// Elastic pools that follow a DTU pricing model have the following costs associated with them:
//
//  1. Costs based on the number of DTUs that the SQL database has
//  2. Extra backup data costs - this is configured using MSSQLElasticPool.ExtraDataStorageGB
//  3. Long term data backup costs - this is configured using MSSQLElasticPool.LongTermRetentionStorageGB
//
// Elastic pools that follow a vCore pricing model have the following costs associated with them:
//
//  1. Costs based on the number of vCores the resource has
//  2. Additional charge for SQL Server licensing based on vCores amount
//  3. Charges for storage used
//  4. Charges for long term data backup - this is configured using MSSQLElasticPool.LongTermRetentionStorageGB
//
// This method is called after the resource is initialized by an IaC provider. MSSQLElasticPool is used by both mssql_elasticpool
// and sql_elasticpool Terraform catalog.
func (r *MSSQLElasticPool) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: r.costComponents(),
	}
}

func (r *MSSQLElasticPool) costComponents() []*engine.LineItem {
	s := strings.ToLower(r.SKU)
	if s == "basicpool" || s == "standardpool" || s == "premiumpool" {
		return r.dtuCostComponents()
	}

	return r.vCoreCostComponents()
}

func (r *MSSQLElasticPool) dtuCostComponents() []*engine.LineItem {
	productName := fmt.Sprintf("SQL Database Elastic Pool - %s", r.Tier)

	var dtuCapacity int64
	if r.DTUCapacity != nil {
		dtuCapacity = *r.DTUCapacity
	}

	costComponents := []*engine.LineItem{
		{
			Name: fmt.Sprintf("Compute (%s, %d DTUs)", r.Tier, dtuCapacity),
			Unit: "hours",
			// This is a bit of a hack, but the Azure pricing API returns the price per day
			// and the Azure pricing calculator uses 730 hours to show the cost
			// so we need to convert the price per day to price per hour.
			UnitMultiplier:  engine.DayToMonthUnitMultiplier,
			MonthlyQuantity: decimalPtr(engine.DaysInMonth),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr(vendorName),
				Region:        strPtr(r.Region),
				Service:       strPtr("SQL Database"),
				ProductFamily: strPtr("Databases"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr(productName)},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("%d DTU Pack", dtuCapacity))},
					{Key: "meterName", Value: strPtr("eDTUs")},
				},
			},
			PriceFilter: priceFilterConsumption,
		},
	}

	var extraStorageGB float64

	if strings.ToLower(r.Tier) == "standard" && r.MaxSizeGB != nil {
		includedStorageGB := float64(dtuCapacity)
		extraStorageGB = *r.MaxSizeGB - includedStorageGB
	} else if strings.ToLower(r.Tier) == "premium" && r.MaxSizeGB != nil {
		includedStorageGB, ok := mssqlElasticPoolPremiumDTUIncludedStorage[int(*r.DTUCapacity)]
		if ok {
			extraStorageGB = *r.MaxSizeGB - includedStorageGB
		}
	}

	if extraStorageGB > 0 {
		costComponents = append(costComponents, r.extraDataStorageCostComponent(extraStorageGB))
	}

	return costComponents
}

func (r *MSSQLElasticPool) vCoreCostComponents() []*engine.LineItem {
	costComponents := r.computeHoursCostComponents()

	if strings.ToLower(r.LicenseType) == "licenseincluded" {
		costComponents = append(costComponents, r.sqlLicenseCostComponent())
	}

	costComponents = append(costComponents, r.storageCostComponent())

	return costComponents
}

func (r *MSSQLElasticPool) computeHoursCostComponents() []*engine.LineItem {
	var cores int64
	if r.Cores != nil {
		cores = *r.Cores
	}

	productNameRegex := fmt.Sprintf("/%s - %s/", r.Tier, r.Family)
	name := fmt.Sprintf("Compute (%s, %d vCore)", r.SKU, cores)

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
			PriceFilter: priceFilterConsumption,
		},
	}

	// Zone redundancy is free for premium and business critical tiers
	if strings.EqualFold(r.Tier, sqlGeneralPurposeTier) && r.ZoneRedundant {
		costComponents = append(costComponents, &engine.LineItem{
			Name:           fmt.Sprintf("Zone redundancy (%s, %d vCore)", r.SKU, cores),
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

func (r *MSSQLElasticPool) extraDataStorageCostComponent(extraStorageGB float64) *engine.LineItem {
	return mssqlExtraDataStorageCostComponent(r.Region, r.Tier, extraStorageGB)
}

func (r *MSSQLElasticPool) sqlLicenseCostComponent() *engine.LineItem {
	return mssqlLicenseCostComponent(r.Region, r.Cores, r.Tier)
}

func (r *MSSQLElasticPool) storageCostComponent() *engine.LineItem {
	return mssqlStorageCostComponent(r.Region, r.Tier, r.ZoneRedundant, r.MaxSizeGB)
}

func (r *MSSQLElasticPool) productFilter(filters []*engine.AttributeMatch) *engine.ProductSelector {
	return mssqlProductFilter(r.Region, filters)
}
