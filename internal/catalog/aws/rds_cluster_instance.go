package aws

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	eng "github.com/c3xdev/c3x/internal/engine"
)

type RDSClusterInstance struct {
	Address                                      string
	Region                                       string
	InstanceClass                                string
	Engine                                       string
	Version                                      string
	IOOptimized                                  bool
	PerformanceInsightsEnabled                   bool
	PerformanceInsightsLongTermRetention         bool
	MonthlyCPUCreditHrs                          *int64   `c3x_usage:"monthly_cpu_credit_hrs"`
	VCPUCount                                    *int64   `c3x_usage:"vcpu_count"`
	MonthlyAdditionalPerformanceInsightsRequests *int64   `c3x_usage:"monthly_additional_performance_insights_requests"`
	ReservedInstanceTerm                         *string  `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption                *string  `c3x_usage:"reserved_instance_payment_option"`
	CapacityUnitsPerHr                           *float64 `c3x_usage:"capacity_units_per_hr"`
}

func (r *RDSClusterInstance) CoreType() string {
	return "RDSClusterInstance"
}

func (r *RDSClusterInstance) UsageSchema() []*eng.ConsumptionField {
	return []*eng.ConsumptionField{
		{Key: "monthly_cpu_credit_hrs", ValueType: eng.Int64, DefaultValue: 0},
		{Key: "vcpu_count", ValueType: eng.Int64, DefaultValue: 0},
		{Key: "monthly_additional_performance_insights_requests", ValueType: eng.Int64, DefaultValue: 0},
		{Key: "reserved_instance_term", DefaultValue: "", ValueType: eng.String},
		{Key: "reserved_instance_payment_option", DefaultValue: "", ValueType: eng.String},
		{Key: "capacity_units_per_hr", ValueType: eng.Float64, DefaultValue: 0},
	}
}

func (r *RDSClusterInstance) PopulateUsage(u *eng.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *RDSClusterInstance) BuildResource() *eng.Estimate {
	databaseEngine := r.databaseEngineValue()

	costComponents := []*eng.LineItem{}
	isServerless := strings.EqualFold(r.InstanceClass, "db.serverless")
	if isServerless {
		costComponents = append(costComponents, r.auroraServerlessV2CostComponent(databaseEngine))
	} else {
		costComponents = append(costComponents, r.dbInstanceCostComponent(databaseEngine))
	}

	if instanceFamily := getBurstableInstanceFamily([]string{"db.t3", "db.t4g"}, r.InstanceClass); instanceFamily != "" {
		instanceCPUCreditHours := decimal.Zero
		if r.MonthlyCPUCreditHrs != nil {
			instanceCPUCreditHours = decimal.NewFromInt(*r.MonthlyCPUCreditHrs)
		}

		instanceVCPUCount := decimal.Zero
		if r.VCPUCount != nil {
			// VCPU count has been set explicitly
			instanceVCPUCount = decimal.NewFromInt(*r.VCPUCount)
		} else if count, ok := InstanceTypeToVCPU[strings.TrimPrefix(r.InstanceClass, "db.")]; ok {
			// We were able to lookup thing VCPU count
			instanceVCPUCount = decimal.NewFromInt(count)
		}

		if instanceCPUCreditHours.GreaterThan(decimal.NewFromInt(0)) {
			cpuCreditQuantity := instanceVCPUCount.Mul(instanceCPUCreditHours)
			costComponents = append(costComponents, r.cpuCreditsCostComponent(databaseEngine, instanceFamily, cpuCreditQuantity))
		}
	}
	if r.PerformanceInsightsEnabled {
		if r.PerformanceInsightsLongTermRetention {
			costComponents = append(costComponents, performanceInsightsLongTermRetentionCostComponent(r.Region, r.InstanceClass, databaseEngine, isServerless, r.CapacityUnitsPerHr))
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
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *RDSClusterInstance) databaseEngineValue() string {
	if r.Engine == "aurora-postgresql" {
		return "Aurora PostgreSQL"
	}

	return "Aurora MySQL"
}

func (r *RDSClusterInstance) dbInstanceCostComponent(databaseEngine string) *eng.LineItem {
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

	// Example usage types for Aurora
	// InstanceUsage:db.t3.medium
	// InstanceUsageIOOptimized:db.t3.medium
	// EU-InstanceUsage:db.t3.medium
	// EU-InstanceUsageIOOptimized:db.t3.medium
	usageTypeFilter := "/InstanceUsage:/"
	if r.IOOptimized {
		usageTypeFilter = "/InstanceUsageIOOptimized:/"
	}

	return &eng.LineItem{
		Name:           fmt.Sprintf("Database instance (%s, %s)", purchaseOptionLabel, r.InstanceClass),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRDS"),
			ProductFamily: strPtr("Database Instance"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "instanceType", Value: strPtr(r.InstanceClass)},
				{Key: "databaseEngine", Value: strPtr(databaseEngine)},
				{Key: "usagetype", ValueRegex: strPtr(usageTypeFilter)},
			},
		},
		PriceFilter: priceFilter,
	}
}

func (r *RDSClusterInstance) auroraServerlessV2CostComponent(databaseEngine string) *eng.LineItem {
	var auroraCapacityUnits *decimal.Decimal
	if r.CapacityUnitsPerHr != nil {
		auroraCapacityUnits = decimalPtr(decimal.NewFromFloat(*r.CapacityUnitsPerHr))
	}

	label := "Aurora serverless v2"
	usageType := "Aurora:ServerlessV2Usage$"
	if r.IOOptimized {
		label = "Aurora serverless v2 (I/O-optimized)"
		usageType = "Aurora:ServerlessV2IOOptimizedUsage$"
	}

	return &eng.LineItem{
		Name:           label,
		Unit:           "ACU-hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: auroraCapacityUnits,
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRDS"),
			ProductFamily: strPtr("ServerlessV2"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "databaseEngine", Value: strPtr(databaseEngine)},
				{Key: "usagetype", ValueRegex: regexPtr(usageType)},
			},
		},
		UsageBased: true,
	}
}

func (r *RDSClusterInstance) cpuCreditsCostComponent(databaseEngine, instanceFamily string, vCPUCount decimal.Decimal) *eng.LineItem {
	return &eng.LineItem{
		Name:            "CPU credits",
		Unit:            "vCPU-hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: &vCPUCount,
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRDS"),
			ProductFamily: strPtr("CPU Credits"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "databaseEngine", Value: strPtr(databaseEngine)},
				{Key: "usagetype", ValueRegex: regexPtr("CPUCredits:" + instanceFamily + "$")},
			},
		},
		UsageBased: true,
	}
}
