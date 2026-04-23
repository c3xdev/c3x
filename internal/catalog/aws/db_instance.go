package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	eng "github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DBInstance struct {
	Address                                      string
	Region                                       string
	LicenseModel                                 string
	StorageType                                  string
	BackupRetentionPeriod                        int64
	IOOptimized                                  bool
	PerformanceInsightsEnabled                   bool
	PerformanceInsightsLongTermRetention         bool
	MultiAZ                                      bool
	InstanceClass                                string
	Engine                                       string
	Version                                      string
	IOPS                                         float64
	AllocatedStorageGB                           *float64
	MonthlyStandardIORequests                    *int64   `c3x_usage:"monthly_standard_io_requests"`
	AdditionalBackupStorageGB                    *float64 `c3x_usage:"additional_backup_storage_gb"`
	MonthlyAdditionalPerformanceInsightsRequests *int64   `c3x_usage:"monthly_additional_performance_insights_requests"`
	ReservedInstanceTerm                         *string  `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption                *string  `c3x_usage:"reserved_instance_payment_option"`
}

func (r *DBInstance) CoreType() string {
	return "DBInstance"
}

func (r *DBInstance) UsageSchema() []*eng.ConsumptionField {
	return DBInstanceUsageSchema
}

var DBInstanceUsageSchema = []*eng.ConsumptionField{
	{Key: "monthly_standard_io_requests", ValueType: eng.Int64, DefaultValue: 0},
	{Key: "additional_backup_storage_gb", ValueType: eng.Float64, DefaultValue: 0},
	{Key: "monthly_additional_performance_insights_requests", ValueType: eng.Int64, DefaultValue: 0},
	{Key: "reserved_instance_term", DefaultValue: "", ValueType: eng.String},
	{Key: "reserved_instance_payment_option", DefaultValue: "", ValueType: eng.String},
}

func (r *DBInstance) PopulateUsage(u *eng.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DBInstance) BuildResource() *eng.Estimate {
	deploymentOption := "Single-AZ"
	if r.MultiAZ {
		deploymentOption = "Multi-AZ"
	}

	var monthlyIORequests *decimal.Decimal
	if r.MonthlyStandardIORequests != nil {
		monthlyIORequests = decimalPtr(decimal.NewFromInt(*r.MonthlyStandardIORequests))
	}

	var databaseEngine string
	switch strings.ToLower(r.Engine) {
	case "postgres":
		databaseEngine = "PostgreSQL"
	case "mysql":
		databaseEngine = "MySQL"
	case "mariadb":
		databaseEngine = "MariaDB"
	case "aurora", "aurora-mysql":
		databaseEngine = "Aurora MySQL"
	case "aurora-postgresql":
		databaseEngine = "Aurora PostgreSQL"
	case "oracle-se", "oracle-se1", "oracle-se2", "oracle-se2-cdb", "oracle-ee", "oracle-ee-cdb":
		databaseEngine = "Oracle"
	case "sqlserver-ex", "sqlserver-web", "sqlserver-se", "sqlserver-ee":
		databaseEngine = "SQL Server"
	}

	var databaseEdition string
	switch strings.ToLower(r.Engine) {
	case "oracle-se", "sqlserver-se":
		databaseEdition = "Standard"
	case "oracle-se1":
		databaseEdition = "Standard One"
	case "oracle-se2", "oracle-se2-cdb":
		databaseEdition = "Standard Two"
	case "oracle-ee", "oracle-ee-cdb", "sqlserver-ee":
		databaseEdition = "Enterprise"
	case "sqlserver-ex":
		databaseEdition = "Express"
	case "sqlserver-web":
		databaseEdition = "Web"
	}

	var licenseModel string
	engineVal := strings.ToLower(r.Engine)
	if engineVal == "oracle-se1" || engineVal == "oracle-se2" || engineVal == "oracle-se2-cdb" || strings.HasPrefix(engineVal, "sqlserver-") {
		licenseModel = "License included"
	}
	if strings.ToLower(r.LicenseModel) == "bring-your-own-license" {
		licenseModel = "Bring your own license"
	}

	iopsVal := decimal.NewFromFloat(r.IOPS)

	allocatedStorageVal := decimal.NewFromInt(20)
	if r.AllocatedStorageGB != nil {
		allocatedStorageVal = decimal.NewFromFloat(*r.AllocatedStorageGB)
	}

	volumeType := "General Purpose"
	storageName := "Storage (general purpose SSD, gp2)"
	iopsDescription := "RDS Provisioned IOPS"

	storageType := strings.ToLower(r.StorageType)
	switch storageType {
	case "io1":
		volumeType = "Provisioned IOPS"
		storageName = "Storage (provisioned IOPS SSD, io1)"
		if iopsVal.LessThan(decimal.NewFromInt(1000)) {
			iopsVal = decimal.NewFromInt(1000)
		}
		if allocatedStorageVal.LessThan(decimal.NewFromInt(100)) {
			allocatedStorageVal = decimal.NewFromInt(100)
		}
	case "standard":
		volumeType = "Magnetic"
		storageName = "Storage (magnetic)"
	case "gp3":
		volumeType = "General Purpose-GP3"
		storageName = "Storage (general purpose SSD, gp3)"
		iopsDescription = "RDS Provisioned GP3 IOPS"

	}

	instanceAttributeFilters := []*eng.AttributeMatch{
		{Key: "instanceType", Value: strPtr(r.InstanceClass)},
		{Key: "deploymentOption", Value: strPtr(deploymentOption)},
		{Key: "databaseEngine", Value: strPtr(databaseEngine)},
	}
	if databaseEdition != "" {
		instanceAttributeFilters = append(instanceAttributeFilters, &eng.AttributeMatch{
			Key:   "databaseEdition",
			Value: strPtr(databaseEdition),
		})
	}
	if licenseModel != "" {
		instanceAttributeFilters = append(instanceAttributeFilters, &eng.AttributeMatch{
			Key:   "licenseModel",
			Value: strPtr(licenseModel),
		})
	}
	// Note: Oracle products without a "deploymentModel" attribute are standard
	// Oracle instances. Products with deploymentModel "Custom" are for Oracle on
	// custom infrastructure (Outposts). We intentionally do NOT filter on
	// deploymentModel here — the other filters (databaseEdition, licenseModel,
	// deploymentOption) are sufficient to narrow to the correct product.
	if strings.HasPrefix(databaseEngine, "Aurora") {
		// Example usage types for Aurora
		// InstanceUsage:db.t3.medium
		// InstanceUsageIOOptimized:db.t3.medium
		// EU-InstanceUsage:db.t3.medium
		// EU-InstanceUsageIOOptimized:db.t3.medium
		usageTypeFilter := "/InstanceUsage:/"
		if r.IOOptimized {
			usageTypeFilter = "/InstanceUsageIOOptimized:/"
		}

		instanceAttributeFilters = append(instanceAttributeFilters, &eng.AttributeMatch{
			Key:        "usagetype",
			ValueRegex: strPtr(usageTypeFilter),
		})
	}

	purchaseOptionLabel := "on-demand"
	priceFilter := &eng.RateSelector{
		PurchaseOption: strPtr("on_demand"),
	}

	var err error
	if r.ReservedInstanceTerm != nil {
		resolver := &rdsReservationResolver{
			term:          strVal(r.ReservedInstanceTerm),
			paymentOption: strVal(r.ReservedInstancePaymentOption),
		}
		priceFilter, err = resolver.PriceFilter()
		if err != nil {
			logging.Logger.Warn().Msg(err.Error())
		}
		purchaseOptionLabel = "reserved"
	}

	storageFilters := []*eng.AttributeMatch{
		{Key: "deploymentOption", Value: strPtr(deploymentOption)},
		{Key: "databaseEngine", Value: strPtr("Any")},
		{Key: "volumeType", Value: strPtr(volumeType)},
	}

	if storageType == "gp3" {
		if deploymentOption == "Multi-AZ" {
			storageFilters = append(storageFilters, &eng.AttributeMatch{Key: "usagetype", ValueRegex: strPtr("/\\-RDS\\:Multi\\-AZ\\-GP3\\-Storage$/")})
		} else {
			storageFilters = append(storageFilters, &eng.AttributeMatch{Key: "usagetype", ValueRegex: strPtr("/\\-RDS\\:GP3\\-Storage$/")})
		}
	}

	costComponents := []*eng.LineItem{
		{
			Name:           fmt.Sprintf("Database instance (%s, %s, %s)", purchaseOptionLabel, deploymentOption, r.InstanceClass),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &eng.ProductSelector{
				VendorName:       strPtr("aws"),
				Region:           strPtr(r.Region),
				Service:          strPtr("AmazonRDS"),
				ProductFamily:    strPtr("Database Instance"),
				AttributeFilters: instanceAttributeFilters,
			},
			PriceFilter: priceFilter,
		},
		{
			Name:            storageName,
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: &allocatedStorageVal,
			ProductFilter: &eng.ProductSelector{
				VendorName:       strPtr("aws"),
				Region:           strPtr(r.Region),
				Service:          strPtr("AmazonRDS"),
				ProductFamily:    strPtr("Database Storage"),
				AttributeFilters: storageFilters,
			},
		},
	}

	if strings.ToLower(volumeType) == "magnetic" {
		costComponents = append(costComponents, &eng.LineItem{
			Name:            "I/O requests",
			Unit:            "1M requests",
			UnitMultiplier:  decimal.NewFromInt(1000000),
			MonthlyQuantity: monthlyIORequests,
			ProductFilter: &eng.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonRDS"),
				ProductFamily: strPtr("System Operation"),
				AttributeFilters: []*eng.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/RDS:StorageIOUsage/i")},
				},
			},
			UsageBased: true,
		})
	}

	if storageType == "io1" || storageType == "gp3" {
		if storageType == "gp3" {
			// For GP3 Storage volumes, all IOPS and throughput use below the baseline is
			// included at no additional charge. For volumes below 400 GiB of allocated
			// storage, the baseline provisioned IOPS is 3,000 and baseline throughput is 125
			// MiBps. Volumes of 400 GiB and above, baseline provisioned IOPS is 12,000 and
			// baseline throughput is 500 MiBps. There is an additional charge for
			// provisioned IOPS and throughput above baseline.
			baseline := decimal.NewFromInt(3000)
			baselineStr := "3,000"
			if allocatedStorageVal.GreaterThanOrEqual(decimal.NewFromInt(400)) {
				baseline = decimal.NewFromInt(12000)
				baselineStr = "12,000"
			}

			if iopsVal.GreaterThan(baseline) {
				over := iopsVal.Sub(baseline)

				usageType := strPtr("/\\-RDS\\:GP3\\-PIOPS$/")
				if deploymentOption == "Multi-AZ" {
					usageType = strPtr("/\\-RDS\\:Multi\\-AZ\\-GP3\\-PIOPS$/")
				}

				costComponents = append(costComponents, &eng.LineItem{
					Name:            fmt.Sprintf("Provisioned GP3 IOPS (above %s)", baselineStr),
					Unit:            "IOPS",
					UnitMultiplier:  decimal.NewFromInt(1),
					MonthlyQuantity: &over,
					ProductFilter: &eng.ProductSelector{
						VendorName:    strPtr("aws"),
						Region:        strPtr(r.Region),
						Service:       strPtr("AmazonRDS"),
						ProductFamily: strPtr("Provisioned IOPS"),
						AttributeFilters: []*eng.AttributeMatch{
							{Key: "deploymentOption", Value: strPtr(deploymentOption)},
							{Key: "groupDescription", Value: strPtr(iopsDescription)},
							{Key: "databaseEngine", Value: strPtr("Any")},
							{Key: "usagetype", ValueRegex: usageType},
						},
					},
				})
			}
		} else {
			costComponents = append(costComponents, &eng.LineItem{
				Name:            "Provisioned IOPS",
				Unit:            "IOPS",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: &iopsVal,
				ProductFilter: &eng.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonRDS"),
					ProductFamily: strPtr("Provisioned IOPS"),
					AttributeFilters: []*eng.AttributeMatch{
						{Key: "deploymentOption", Value: strPtr(deploymentOption)},
						{Key: "groupDescription", Value: strPtr(iopsDescription)},
						{Key: "databaseEngine", Value: strPtr("Any")},
					},
				},
			})
		}

	}

	var backupStorageGB *decimal.Decimal
	if r.AdditionalBackupStorageGB != nil {
		backupStorageGB = decimalPtr(decimal.NewFromFloat(*r.AdditionalBackupStorageGB))
	}

	if r.BackupRetentionPeriod > 0 || (backupStorageGB != nil && backupStorageGB.GreaterThan(decimal.Zero)) {
		backupStorageDBEngine := "Any"
		attrFilters := []*eng.AttributeMatch{
			{Key: "databaseEngine", Value: strPtr(backupStorageDBEngine)},
			{Key: "usagetype", ValueRegex: regexPtr("RDS:ChargedBackupUsage$")},
			{Key: "engineCode", ValueRegex: regexPtr("[0-9]+")},
			{Key: "operation", Value: strPtr("")},
		}

		if strings.HasPrefix(databaseEngine, "Aurora") {
			backupStorageDBEngine = databaseEngine
			attrFilters = []*eng.AttributeMatch{
				{Key: "databaseEngine", Value: strPtr(backupStorageDBEngine)},
				{Key: "usagetype", ValueRegex: regexPtr("Aurora:BackupUsage$")},
				{Key: "engineCode", ValueRegex: regexPtr("[0-9]+")},
			}
		}

		costComponents = append(costComponents, &eng.LineItem{
			Name:            "Additional backup storage",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: backupStorageGB,
			ProductFilter: &eng.ProductSelector{
				VendorName:       strPtr("aws"),
				Region:           strPtr(r.Region),
				Service:          strPtr("AmazonRDS"),
				ProductFamily:    strPtr("Storage Snapshot"),
				AttributeFilters: attrFilters,
			},
			UsageBased: true,
		})
	}

	if r.PerformanceInsightsEnabled {
		if r.PerformanceInsightsLongTermRetention {
			costComponents = append(costComponents, performanceInsightsLongTermRetentionCostComponent(r.Region, r.InstanceClass, databaseEngine, false, nil))
		}

		if r.MonthlyAdditionalPerformanceInsightsRequests == nil || *r.MonthlyAdditionalPerformanceInsightsRequests > 0 {
			costComponents = append(costComponents,
				performanceInsightsAPIRequestCostComponent(r.Region, r.MonthlyAdditionalPerformanceInsightsRequests))
		}
	}

	extendedSupport := extendedSupportCostComponent(r.Version, r.Region, r.Engine, r.InstanceClass)
	if extendedSupport != nil {
		costComponents = append(costComponents, extendedSupport)
	}

	return &eng.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    DBInstanceUsageSchema,
	}
}
func performanceInsightsLongTermRetentionCostComponent(region, instanceClass, dbEngine string, isServerless bool, capacityUnits *float64) *eng.LineItem {

	if isServerless {
		auroraCapacityUnits := decimal.Zero
		if capacityUnits != nil {
			auroraCapacityUnits = decimal.NewFromFloat(*capacityUnits)
		}
		return &eng.LineItem{
			Name:            "Performance Insights Long Term Retention (serverless)",
			Unit:            "ACUs",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: &auroraCapacityUnits,
			ProductFilter: &eng.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonRDS"),
				ProductFamily: strPtr("Performance Insights"),
				AttributeFilters: []*eng.AttributeMatch{
					{
						Key:        "usagetype",
						ValueRegex: regexPtr("PI_LTR_FMR:Serverless$"),
					},
					{
						Key:   "databaseEngine",
						Value: &dbEngine,
					},
				},
			},
		}
	}

	instanceType := strings.TrimPrefix(instanceClass, "db.")

	vCPUCount := decimal.Zero
	if count, ok := InstanceTypeToVCPU[instanceType]; ok {
		// We were able to lookup thing VCPU count
		vCPUCount = decimal.NewFromInt(count)
	}

	var instanceFamily string
	split := strings.SplitN(instanceType, ".", 2)
	if len(split) > 0 {
		instanceFamily = split[0]
	}

	return &eng.LineItem{
		Name:            fmt.Sprintf("Performance Insights Long Term Retention (%s)", strings.ToLower(instanceClass)),
		Unit:            "vCPU-month",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: &vCPUCount,
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(region),
			Service:       strPtr("AmazonRDS"),
			ProductFamily: strPtr("Performance Insights"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("PI_LTR:" + strings.ToUpper(instanceFamily) + "$")},
			},
		},
	}
}

func performanceInsightsAPIRequestCostComponent(region string, additionalRequests *int64) *eng.LineItem {
	return &eng.LineItem{
		Name:            "Performance Insights API",
		Unit:            "1000 requests",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: intPtrToDecimalPtr(additionalRequests),
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(region),
			Service:       strPtr("AmazonRDS"),
			ProductFamily: strPtr("Performance Insights"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("PI_API$")},
			},
		},
		UsageBased: true,
	}
}

// ExtendedSupportDates contains the extended support dates for a specific RDS
// engine version Year1 is the date when the extended support starts, Year 3 is
// the date when the extended increases price.
type ExtendedSupportDates struct {
	UsagetypeVersion string
	Year1            time.Time
	Year3            time.Time
}

// ExtendedSupport contains the extended support dates for a specific RDS eng.
type ExtendedSupport struct {
	Engine   string
	Versions map[string]ExtendedSupportDates
}

// CostComponent returns the cost component for the extended support for the
// given version and date. If the version is not found then it will return nil.
func (s ExtendedSupport) CostComponent(version string, region string, d time.Time, quantity decimal.Decimal) *eng.LineItem {
	matchingVersion := strings.ToLower(version)
	supportDates, ok := s.Versions[matchingVersion]
	if !ok {
		// if the version is not found then it is likely that the
		// version is a minor version, we should try and match the minor
		// version to a major version in the map. This is done by
		// progressively removing the last part of the version until
		// we find a match.
		parts := strings.Split(version, ".")
		for i := len(parts) - 1; i > 0; i-- {
			matchingVersion = strings.Join(parts[:i], ".")
			supportDates, ok = s.Versions[matchingVersion]
			if ok {
				break
			}
		}

		if !ok {
			return nil
		}
	}

	usagetypeVersion := supportDates.UsagetypeVersion
	if usagetypeVersion == "" {
		usagetypeVersion = matchingVersion
	}

	if !supportDates.Year3.IsZero() && d.After(supportDates.Year3) {
		return &eng.LineItem{
			Name:           "Extended support (year 3)",
			Unit:           "vCPU-hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(quantity),
			ProductFilter: &eng.ProductSelector{
				VendorName: strPtr("aws"),
				Region:     strPtr(region),
				Service:    strPtr("AmazonRDS"),
				AttributeFilters: []*eng.AttributeMatch{
					{Key: "usagetype", ValueRegex: regexPtr("ExtendedSupport:Yr3:" + s.Engine + usagetypeVersion)},
				},
			},
		}
	}

	if d.After(supportDates.Year1) {
		return &eng.LineItem{
			Name:           "Extended support (year 1)",
			Unit:           "vCPU-hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(quantity),
			ProductFilter: &eng.ProductSelector{
				VendorName: strPtr("aws"),
				Region:     strPtr(region),
				Service:    strPtr("AmazonRDS"),
				AttributeFilters: []*eng.AttributeMatch{
					{Key: "usagetype", ValueRegex: regexPtr("ExtendedSupport:Yr1-Yr2:" + s.Engine + usagetypeVersion)},
				},
			},
		}
	}

	return nil
}

var (
	// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MySQL.Concepts.VersionMgmt.html#MySQL.Concepts.VersionMgmt.ReleaseCalendar
	mysqlExtendedSupport = ExtendedSupport{
		Engine: "MySQL",
		Versions: map[string]ExtendedSupportDates{
			"5.7": {Year1: time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"8":   {Year1: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2028, time.August, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	// https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Aurora.VersionPolicy.html#Aurora.VersionPolicy.MajorVersions
	mysqlAuroraExtendedSupport = ExtendedSupport{
		Engine: "AuroraMySQL",
		Versions: map[string]ExtendedSupportDates{
			"5.7": {UsagetypeVersion: "2", Year1: time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC)}, // Year3 is zero because it's N/A
			"8":   {UsagetypeVersion: "3", Year1: time.Date(2027, time.May, 1, 0, 0, 0, 0, time.UTC)},      // Year3 is zero because it's N/A
		},
	}

	// https://docs.aws.amazon.com/AmazonRDS/latest/PostgreSQLReleaseNotes/postgresql-release-calendar.html#Release.Calendar
	postgresExtendedSupport = ExtendedSupport{
		Engine: "PostgreSQL",
		Versions: map[string]ExtendedSupportDates{
			"11": {Year1: time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)},
			"12": {Year1: time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2027, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"13": {Year1: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"14": {Year1: time.Date(2027, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2029, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"15": {Year1: time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2030, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"16": {Year1: time.Date(2029, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2031, time.March, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	// https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Aurora.VersionPolicy.html#Aurora.VersionPolicy.MajorVersions
	postgresAuroraExtendedSupport = ExtendedSupport{
		Engine: "AuroraPostgreSQL",
		Versions: map[string]ExtendedSupportDates{
			"11": {Year1: time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)},
			"12": {Year1: time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2027, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"13": {Year1: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"14": {Year1: time.Date(2027, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2029, time.March, 1, 0, 0, 0, 0, time.UTC)},
			"15": {Year1: time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC), Year3: time.Date(2030, time.March, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	// Today is the date used to determine extended support pricing tiers.
	// It can be overridden in tests to make pricing deterministic.
	Today = time.Now()
)

func extendedSupportCostComponent(version string, region string, engine string, instanceType string) *eng.LineItem {
	if version == "" {
		return nil
	}

	vCPUCount := decimal.NewFromInt(1)
	if count, ok := InstanceTypeToVCPU[strings.TrimPrefix(instanceType, "db.")]; ok {
		// We were able to lookup thing VCPU count
		vCPUCount = decimal.NewFromInt(count)
	}

	switch engine {
	case "postgres":
		return postgresExtendedSupport.CostComponent(version, region, Today, vCPUCount)
	case "mysql":
		return mysqlExtendedSupport.CostComponent(version, region, Today, vCPUCount)
	case "aurora-postgresql":
		return postgresAuroraExtendedSupport.CostComponent(version, region, Today, vCPUCount)
	case "aurora", "aurora-mysql":
		return mysqlAuroraExtendedSupport.CostComponent(version, region, Today, vCPUCount)
	}

	return nil
}
