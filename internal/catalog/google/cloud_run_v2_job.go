package google

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// CloudRunV2Job represents a resource that references a container image which is run to completion.
type CloudRunV2Job struct {
	Address              string
	Region               string
	CpuLimit             int64
	MemoryLimit          int64
	TaskCount            int64
	MonthlyJobExecutions *int64   `c3x_usage:"monthly_job_executions"`
	AvgTaskExecutionMins *float64 `c3x_usage:"average_task_execution_mins"`
}

// CoreType returns the name of this resource type
func (r *CloudRunV2Job) CoreType() string {
	return "CloudRunV2Job"
}

// UsageSchema defines a list which represents the usage schema of CloudRunV2Job.
func (r *CloudRunV2Job) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_job_executions", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "average_task_execution_mins", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *CloudRunV2Job) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudRunV2Job) BuildResource() *engine.Estimate {
	regionTier := GetRegionTier(r.Region)
	cpuName := "CPU allocation time"
	memoryName := "Memory allocation time"
	if regionTier == "Tier 2" {
		cpuName = "CPU allocation time (tier 2)"
		memoryName = "Memory allocation time (tier 2)"
	}

	costComponents := []*engine.LineItem{
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
					{Key: "description", Value: strPtr(fmt.Sprintf("Jobs CPU in %s", r.Region))},
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
					{Key: "description", Value: strPtr(fmt.Sprintf("Jobs Memory in %s", r.Region))},
				},
			},
		},
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *CloudRunV2Job) calculateCpuSeconds() *decimal.Decimal {
	if r.AvgTaskExecutionMins == nil || r.MonthlyJobExecutions == nil {
		return nil
	}

	seconds := decimal.NewFromFloat(*r.AvgTaskExecutionMins * 60)
	return decimalPtr(decimal.NewFromInt(*r.MonthlyJobExecutions).Mul(decimal.NewFromInt(r.TaskCount)).Mul(seconds).Mul(decimal.NewFromInt(r.CpuLimit)))
}

func (r *CloudRunV2Job) calculateGBSeconds() *decimal.Decimal {
	if r.AvgTaskExecutionMins == nil || r.MonthlyJobExecutions == nil {
		return nil
	}

	seconds := decimal.NewFromFloat(*r.AvgTaskExecutionMins * 60)
	gb := decimal.NewFromInt(r.MemoryLimit).Div(decimal.NewFromInt(1024 * 1024 * 1024))
	return decimalPtr(decimal.NewFromInt(*r.MonthlyJobExecutions).Mul(decimal.NewFromInt(r.TaskCount)).Mul(seconds).Mul(gb))
}
