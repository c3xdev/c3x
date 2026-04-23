package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

const (
	logAnalyticsServiceName = "Log Analytics"
	azureMonitorServiceName = "Azure Monitor"
	governanceProductFamily = "Management and Governance"

	skuCapacityReservation = "CapacityReservation"
	skuPerGB2018           = "PerGB2018"
	skuFree                = "Free"
	skuFilterAnalyticsLogs = "Analytics Logs"
	skuBasicLogsIngest     = "Basic Logs"
	skuBasicLogsSearch     = "Search Queries"
	skuArchive             = "Data Archive"
	skuArchiveRestore      = "Data Restore"
	skuArchiveSearch       = "Search Jobs"

	logRetentionFreeTierLimit = 30
)

var (
	// unsupportedLegacySkus represents skus that C3X doesn't support because these skus are
	// legacy pricing tiers: https://docs.microsoft.com/en-us/azure/azure-monitor//logs/manage-cost-storage#legacy-pricing-tiers
	unsupportedLegacySkus = map[string]struct{}{
		"unlimited": {},
		"standard":  {},
		"premium":   {},
		"pernode":   {},
	}

	// validCommitmentTiers is a lookup map of valid data ingestion tiers.
	validCommitmentTiers = map[int64]struct{}{
		100:  {},
		200:  {},
		300:  {},
		400:  {},
		500:  {},
		1000: {},
		2000: {},
		5000: {},
	}

	// commitmentTiers list of valid data ingestion tiers, this can be used
	// to transform invalid tier into a billable unit.
	commitmentTiers = []int64{
		100,
		200,
		300,
		400,
		500,
		1000,
		2000,
		5000,
	}
)

// LogAnalyticsWorkspace struct represents an Azure Monitor log workspace. A workspace consolidates data
// from multiple sources into a single data lake. A workspace defines:
//
//  1. The geographic location of the data.
//  2. Access rights that define which users can access data.
//  3. Configuration settings such as the pricing tier and data retention.
//
// Resource information: https://azure.microsoft.com/en-gb/services/monitor/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/monitor/
type LogAnalyticsWorkspace struct {
	Address string
	Region  string
	SKU     string

	ReservationCapacityInGBPerDay int64
	RetentionInDays               int64
	SentinelEnabled               bool

	MonthlyArchivedDataGB               *float64 `c3x_usage:"monthly_archive_data_gb"`
	MonthlyArchivedDataRestoredGB       *float64 `c3x_usage:"monthly_archive_data_restored_gb"`
	MonthlyArchivedDataSearchedGB       *float64 `c3x_usage:"monthly_archive_data_searched_gb"`
	MonthlyBasicLogDataIngestionGB      *float64 `c3x_usage:"monthly_basic_log_data_ingestion_gb"`
	MonthlyBasicLogSearchGB             *float64 `c3x_usage:"monthly_basic_log_search_gb"`
	MonthlyLogDataIngestionGB           *float64 `c3x_usage:"monthly_log_data_ingestion_gb"`
	MonthlyAdditionalLogDataRetentionGB *float64 `c3x_usage:"monthly_additional_log_data_retention_gb"`
	MonthlyLogDataExportGB              *float64 `c3x_usage:"monthly_log_data_export_gb"`
	MonthlySentinelDataIngestionGB      *float64 `c3x_usage:"monthly_sentinel_data_ingestion_gb"`
}

// CoreType returns the name of this resource type
func (r *LogAnalyticsWorkspace) CoreType() string {
	return "LogAnalyticsWorkspace"
}

// UsageSchema defines a list which represents the usage schema of LogAnalyticsWorkspace.
func (r *LogAnalyticsWorkspace) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{
			Key:          "monthly_archived_data_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_archived_data_restored_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_archived_data_searched_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_basic_log_data_ingestion_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_basic_log_search_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_log_data_ingestion_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_additional_log_data_retention_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_log_data_export_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
		{
			Key:          "monthly_sentinel_data_ingestion_gb",
			DefaultValue: 0,
			ValueType:    engine.Float64,
		},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the LogAnalyticsWorkspace.
// It uses the `c3x_usage` struct tags to populate data into the LogAnalyticsWorkspace.
func (r *LogAnalyticsWorkspace) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid LogAnalyticsWorkspace struct.
// The returned engine.Estimate can have 9 potential engine.LineItem associated with it:
//
//  1. Log data ingestion, which can be either:
//     a) Pay-as-you-go, which is only valid for a sku of PerGB2018 and uses a usage param
//     b) Billed per commitment tiers, which is only valid for a sku of CapacityReservation
//  2. Log retention, which is free up to 31 days. Data retained beyond these no-charge periods
//     will be charged for each GB of data retained for a month (pro-rated daily).
//  3. Data export, which is billed per monthly GB exported and is defined from a usage param.
//  4. Sentinel data ingestion if Sentinel usage is detected.
//  5. Basic log data ingestion, which is a less expensive of tier for "ingesting and storing
//     high-volume verbose logs in your Log Analytics workspace for debugging, troubleshooting,
//     and auditing, but not for analytics and alerts."
//  6. Basic log search, which is billed per monthly GB of basic log data queried.
//  7. Archive data, which is billed per monthly GB of archived data
//  8. Archive restore, which is billed per monthly GB of archived data restored
//  9. Archive search, which is billed per monthly GB of archived data searched
//
// Outside the above rules - if the workspace has sku of Free we return as a free resource & if the workspace sku
// is in a list of unsupported skus then we mark as skipped with a warning.
func (r *LogAnalyticsWorkspace) BuildResource() *engine.Estimate {
	if r.SKU == skuFree {
		return &engine.Estimate{
			Name:        r.Address,
			IsSkipped:   true,
			NoPrice:     true,
			UsageSchema: r.UsageSchema(),
		}
	}

	if _, ok := unsupportedLegacySkus[strings.ToLower(r.SKU)]; ok {
		logging.Logger.Warn().Msgf("skipping %s as it uses legacy pricing options", r.Address)

		return &engine.Estimate{
			Name:        r.Address,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	var costComponents []*engine.LineItem

	if r.SKU == skuPerGB2018 {
		costComponents = append(costComponents, r.logDataIngestion("Log data ingestion", r.MonthlyLogDataIngestionGB))

		if r.SentinelEnabled {
			costComponents = append(costComponents, r.logDataIngestion("Sentinel data ingestion", r.MonthlySentinelDataIngestionGB))
		}
	}

	if r.SKU == skuCapacityReservation && r.ReservationCapacityInGBPerDay > 0 {
		costComponents = append(costComponents, r.logDataIngestionFromCapacityReservation("Log data ingestion"))

		if r.SentinelEnabled {
			costComponents = append(costComponents, r.logDataIngestionFromCapacityReservation("Sentinel data ingestion"))
		}
	}

	if r.RetentionInDays > logRetentionFreeTierLimit {
		costComponents = append(costComponents, r.logDataRetention())
	}

	costComponents = append(costComponents, r.logDataExport())
	costComponents = append(costComponents, r.basicLogIngestion())
	costComponents = append(costComponents, r.basicLogSearch())
	costComponents = append(costComponents, r.archiveData())
	costComponents = append(costComponents, r.archiveDataRestore())
	costComponents = append(costComponents, r.archiveDataSearch())

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *LogAnalyticsWorkspace) logDataIngestionFromCapacityReservation(name string) *engine.LineItem {
	selectedTier := r.ReservationCapacityInGBPerDay

	// if the user has set a reservation capacity tier that doesn't exist (or is a legacy tier) we need
	// to convert this to a valid billable tier.
	if _, ok := validCommitmentTiers[r.ReservationCapacityInGBPerDay]; !ok {
		for i, tier := range commitmentTiers {
			// if the current tier is the final valid commitment tier then
			// set selectedTier as it can't be any other tier.
			if len(commitmentTiers)-1 == i {
				selectedTier = tier
				break
			}

			// if the selectedTier is between two valid tiers, set it as the lower of the two tiers.
			if selectedTier > tier && selectedTier < commitmentTiers[i+1] {
				selectedTier = tier
				break
			}
		}
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            fmt.Sprintf("%d GB (per day)", selectedTier),
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromFloat(30)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(fmt.Sprintf("%d GB Commitment Tier", selectedTier))},
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("^%d GB Commitment Tier", selectedTier))},
			},
		},
		PriceFilter: priceFilterConsumption,
	}
}

func (r *LogAnalyticsWorkspace) logDataIngestion(name string, monthlyData *float64) *engine.LineItem {
	var quantity *decimal.Decimal
	if monthlyData != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*monthlyData))
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(logAnalyticsServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuFilterAnalyticsLogs)},
				{Key: "meterName", ValueRegex: regexPtr("Data Ingestion$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("5"),
		},
		UsageBased: true,
	}
}

func (r *LogAnalyticsWorkspace) logDataRetention() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyAdditionalLogDataRetentionGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyAdditionalLogDataRetentionGB))
	}

	return &engine.LineItem{
		Name:            "Log data retention",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(logAnalyticsServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuFilterAnalyticsLogs)},
				{Key: "meterName", ValueRegex: regexPtr("Data Retention$")},
			},
		},
		PriceFilter: priceFilterConsumption,
		UsageBased:  true,
	}
}

func (r *LogAnalyticsWorkspace) logDataExport() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyLogDataExportGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyLogDataExportGB))
	}

	return &engine.LineItem{
		Name:            "Log data export",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Log Analytics data export")},
				{Key: "meterName", ValueRegex: regexPtr("Data Exported$")},
			},
		},
		PriceFilter: priceFilterConsumption,
		UsageBased:  true,
	}
}

func (r *LogAnalyticsWorkspace) basicLogIngestion() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyBasicLogDataIngestionGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyBasicLogDataIngestionGB))
	}

	return &engine.LineItem{
		Name:            "Basic log data ingestion",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuBasicLogsIngest)},
				{Key: "meterName", Value: strPtr("Basic Logs Data Ingestion")},
			},
		},
		UsageBased: true,
	}
}

func (r *LogAnalyticsWorkspace) basicLogSearch() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyBasicLogSearchGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyBasicLogSearchGB))
	}

	return &engine.LineItem{
		Name:            "Basic log search queries",
		Unit:            "GB searched",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuBasicLogsSearch)},
				{Key: "meterName", Value: strPtr("Search Queries Scanned")},
			},
		},
		UsageBased: true,
	}
}

func (r *LogAnalyticsWorkspace) archiveData() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyArchivedDataGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyArchivedDataGB))
	}

	return &engine.LineItem{
		Name:            "Archive data",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuArchive)},
				{Key: "meterName", Value: strPtr("Data Archive")},
			},
		},
		UsageBased: true,
	}
}

func (r *LogAnalyticsWorkspace) archiveDataRestore() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyArchivedDataRestoredGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyArchivedDataRestoredGB))
	}

	return &engine.LineItem{
		Name:            "Archive data restored",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuArchiveRestore)},
				{Key: "meterName", Value: strPtr("Data Restore")},
			},
		},
		UsageBased: true,
	}
}

func (r *LogAnalyticsWorkspace) archiveDataSearch() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyArchivedDataSearchedGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyArchivedDataSearchedGB))
	}

	return &engine.LineItem{
		Name:            "Archive data searched",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr(vendorName),
			Region:        strPtr(r.Region),
			Service:       strPtr(azureMonitorServiceName),
			ProductFamily: strPtr(governanceProductFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(skuArchiveSearch)},
				{Key: "meterName", Value: strPtr("Search Jobs Scanned")},
			},
		},
		UsageBased: true,
	}
}
