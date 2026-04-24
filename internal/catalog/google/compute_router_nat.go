package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ComputeRouterNAT struct {
	Address                string
	Region                 string
	AssignedVMs            *int64   `c3x_usage:"assigned_vms"`
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *ComputeRouterNAT) CoreType() string {
	return "ComputeRouterNAT"
}

func (r *ComputeRouterNAT) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "assigned_vms", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *ComputeRouterNAT) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeRouterNAT) BuildResource() *engine.Estimate {
	var assignedVMs int64
	if r.AssignedVMs != nil {
		assignedVMs = *r.AssignedVMs
		if assignedVMs > 32 {
			assignedVMs = 32
		}
	}

	var dataProcessedGB *decimal.Decimal
	if r.MonthlyDataProcessedGB != nil {
		dataProcessedGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           "Assigned VMs (first 32)",
				Unit:           "VM-hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(assignedVMs)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Compute Engine"),
					ProductFamily: strPtr("Network"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: strPtr("/NAT Gateway: Uptime charge/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Data processed",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: dataProcessedGB,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Compute Engine"),
					ProductFamily: strPtr("Network"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: strPtr("/NAT Gateway: Data processing charge/")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
