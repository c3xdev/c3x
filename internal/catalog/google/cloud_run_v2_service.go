package google

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// CloudRunService acts as a top-level container that manages a set of configurations and revision
// templates which implement a network service. Service exists to provide a singular abstraction which can
// be access controlled, reasoned about, and which encapsulates software lifecycle decisions such as rollout
// policy and team resource ownership.
//
// Resource information: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service/
type CloudRunService struct {
	Address                       string
	Region                        string
	CpuLimit                      int64
	IsThrottlingEnabled           bool
	MemoryLimit                   int64
	MinInstanceCount              float64
	MonthlyRequests               *int64 `c3x_usage:"monthly_requests"`
	AverageRequestDurationMs      *int64 `c3x_usage:"average_request_duration_ms"`
	ConcurrentRequestsPerInstance *int64 `c3x_usage:"concurrent_requests_per_instance"`
	InstanceHrs                   *int64 `c3x_usage:"instance_hrs"`
}

func (r *CloudRunService) CoreType() string {
	return "CloudRunService"
}

func (r *CloudRunService) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "average_request_duration_ms", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "concurrent_requests_per_instance", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "instance_hrs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *CloudRunService) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudRunService) BuildResource() *engine.Estimate {
	regionTier := GetRegionTier(r.Region)
	cpuName := "CPU allocation Time"
	cpuDesc := "Services CPU (Instance-based billing) in " + r.Region
	memoryName := "Memory allocation time"
	memoryDesc := "Services Memory (Instance-based billing) in " + r.Region

	if regionTier == "Tier 2" {
		cpuName = "CPU allocation time (tier 2)"
		cpuDesc = "Services CPU Tier 2  (Request-based billing)"
		memoryName = "Memory allocation time (tier 2)"
		memoryDesc = "Services Memory Tier 2 (Request-based billing)"
	}

	var costComponents []*engine.LineItem
	if r.IsThrottlingEnabled {
		costComponents = r.throttlingEnabledCostComponents(cpuName, cpuDesc, memoryName, memoryDesc)
	} else {
		costComponents = r.throttlingDisabledCostComponents(cpuName, memoryName)
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *CloudRunService) throttlingEnabledCostComponents(cpuName, cpuDesc, memoryName, memoryDesc string) []*engine.LineItem {
	var requests *decimal.Decimal
	if r.MonthlyRequests != nil {
		requests = decimalPtr(decimal.NewFromInt(*r.MonthlyRequests))
	}

	return []*engine.LineItem{
		{
			Name:            cpuName,
			Unit:            "vCPU-seconds",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.calculateCpuSeconds(),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("gcp"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Cloud Run"),
				ProductFamily: strPtr("ApplicationServices"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "description", Value: strPtr(cpuDesc)},
				},
			},
		},
		{
			Name:            memoryName,
			Unit:            "GiB-seconds",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.calculateGBSeconds(),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("gcp"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Cloud Run"),
				ProductFamily: strPtr("ApplicationServices"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "description", Value: strPtr(memoryDesc)},
				},
			},
		},
		{
			Name:            "Number of requests",
			Unit:            "requests",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: requests,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("gcp"),
				Region:        strPtr("global"),
				Service:       strPtr("Cloud Run"),
				ProductFamily: strPtr("ApplicationServices"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "description", Value: strPtr("Requests")},
				},
			},
			PriceFilter: &engine.RateSelector{
				StartUsageAmount: strPtr("2000000"),
			},
		},
	}
}
func (r *CloudRunService) throttlingDisabledCostComponents(cpuName, memoryName string) []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            cpuName,
			Unit:            "vCPU-seconds",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.calculateCpuSeconds(),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("gcp"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Cloud Run"),
				ProductFamily: strPtr("ApplicationServices"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "description", Value: strPtr(fmt.Sprintf("Services CPU (Instance-based billing) in %s", r.Region))},
				},
			},
		},
		{
			Name:            memoryName,
			Unit:            "GiB-seconds",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.calculateGBSeconds(),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("gcp"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Cloud Run"),
				ProductFamily: strPtr("ApplicationServices"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "description", Value: strPtr(fmt.Sprintf("Services Memory (Instance-based billing) in %s", r.Region))},
				},
			},
		},
	}
}

func (r *CloudRunService) calculateCpuSeconds() *decimal.Decimal {
	if r.IsThrottlingEnabled {
		if r.AverageRequestDurationMs == nil || r.MonthlyRequests == nil || r.ConcurrentRequestsPerInstance == nil {
			return nil
		}

		requestDurationInSeconds := decimal.NewFromInt(*r.AverageRequestDurationMs).Div(decimal.NewFromInt(1000))
		return decimalPtr(decimal.NewFromInt(*r.MonthlyRequests).Mul(requestDurationInSeconds).Div(decimal.NewFromInt(*r.ConcurrentRequestsPerInstance)).Mul(decimal.NewFromInt(r.CpuLimit)))
	}

	if r.InstanceHrs != nil && *r.InstanceHrs > 0 {
		return decimalPtr(decimal.NewFromInt(*r.InstanceHrs * 60 * 60).Mul(decimal.NewFromInt(r.CpuLimit)).Mul(decimal.NewFromFloat(r.MinInstanceCount)))
	}

	return decimalPtr(decimal.NewFromFloat(r.MinInstanceCount * (730 * 60 * 60)).Mul(decimal.NewFromInt(r.CpuLimit)))
}

func (r *CloudRunService) calculateGBSeconds() *decimal.Decimal {
	gb := decimal.NewFromInt(r.MemoryLimit).Div(decimal.NewFromInt(1024 * 1024 * 1024))
	if r.IsThrottlingEnabled {
		if r.AverageRequestDurationMs == nil || r.MonthlyRequests == nil || r.ConcurrentRequestsPerInstance == nil {
			return nil
		}

		requestDurationInSeconds := decimal.NewFromInt(*r.AverageRequestDurationMs).Div(decimal.NewFromInt(1000))
		return decimalPtr(decimal.NewFromInt(*r.MonthlyRequests).Mul(requestDurationInSeconds).Div(decimal.NewFromInt(*r.ConcurrentRequestsPerInstance)).Mul(gb))
	}

	if r.InstanceHrs != nil && *r.InstanceHrs > 0 {
		return decimalPtr(decimal.NewFromInt(*r.InstanceHrs * 60 * 60).Mul(gb).Mul(decimal.NewFromFloat(r.MinInstanceCount)))
	}

	return decimalPtr(decimal.NewFromFloat(r.MinInstanceCount * (730 * 60 * 60)).Mul(gb))
}
