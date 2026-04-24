package aws

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

var defaultVolumeSize = int64(8)

type EBSVolume struct {
	// "required" args that can't really be missing.
	Address    string
	Region     string
	Type       string
	IOPS       int64
	Throughput int64

	// "optional" args that can be empty strings.
	Size *int64

	// "usage" args
	MonthlyStandardIORequests *int64 `c3x_usage:"monthly_standard_io_requests"`
}

func (a *EBSVolume) CoreType() string {
	return "EBSVolume"
}

func (a *EBSVolume) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_standard_io_requests", DefaultValue: 0, ValueType: engine.Int64},
	}
}

func (a *EBSVolume) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(a, u)
}

func (a *EBSVolume) BuildResource() *engine.Estimate {
	if a.Type == "" {
		a.Type = "gp2"
	}

	costComponents := make([]*engine.LineItem, 0)
	subResources := make([]*engine.Estimate, 0)

	costComponents = append(costComponents, a.storageCostComponent())

	if strings.ToLower(a.Type) == "gp3" && a.Throughput > 125 {
		costComponents = append(costComponents, a.provisionedThroughputCostComponent())
	}

	if strings.ToLower(a.Type) == "io1" {
		costComponents = append(costComponents, a.provisionedIOPSCostComponent("EBS:VolumeP-IOPS.piops", a.IOPS))
	} else if strings.ToLower(a.Type) == "io2" {
		costComponents = append(costComponents, a.provisionedIOPSCostComponent("EBS:VolumeP-IOPS.io2$", a.IOPS))
	} else if strings.ToLower(a.Type) == "gp3" && a.IOPS > 3000 {
		costComponents = append(costComponents, a.provisionedIOPSCostComponent("VolumeP-IOPS.gp3", a.IOPS-3000))
	}

	if strings.ToLower(a.Type) == "standard" {
		costComponents = append(costComponents, a.ioRequestsCostComponent())
	}

	return &engine.Estimate{
		Name:           a.Address,
		UsageSchema:    a.UsageSchema(),
		CostComponents: costComponents,
		SubResources:   subResources,
	}
}

func (a *EBSVolume) storageCostComponent() *engine.LineItem {
	size := defaultVolumeSize
	if a.Size != nil {
		size = *a.Size
	}

	var name string
	switch strings.ToLower(a.Type) {
	case "standard":
		name = "Storage (magnetic)"
	case "io1":
		name = "Storage (provisioned IOPS SSD, io1)"
	case "io2":
		name = "Storage (provisioned IOPS SSD, io2)"
	case "st1":
		name = "Storage (throughput optimized HDD, st1)"
	case "sc1":
		name = "Storage (cold HDD, sc1)"
	case "gp3":
		name = "Storage (general purpose SSD, gp3)"
	case "gp2":
		name = "Storage (general purpose SSD, gp2)"
	default:
		name = "Storage (unknown)"
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(size)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(a.Region),
			Service:       strPtr("AmazonEC2"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "volumeApiName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", a.Type))},
			},
		},
	}
}

func (a *EBSVolume) provisionedIOPSCostComponent(usageType string, iops int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Provisioned IOPS",
		Unit:            "IOPS",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(iops)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(a.Region),
			Service:       strPtr("AmazonEC2"),
			ProductFamily: strPtr("System Operation"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "volumeApiName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", a.Type))},
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
			},
		},
	}
}

func (a *EBSVolume) ioRequestsCostComponent() *engine.LineItem {
	var qty *decimal.Decimal
	if a.MonthlyStandardIORequests != nil {
		qty = decimalPtr(decimal.NewFromInt(*a.MonthlyStandardIORequests))
	}

	return &engine.LineItem{
		Name:            "I/O requests",
		Unit:            "1M request",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(a.Region),
			Service:       strPtr("AmazonEC2"),
			ProductFamily: strPtr("System Operation"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "volumeApiName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", a.Type))},
				{Key: "usagetype", ValueRegex: strPtr("/EBS:VolumeIOUsage/i")},
			},
		},
		UsageBased: true,
	}
}

func (a *EBSVolume) provisionedThroughputCostComponent() *engine.LineItem {
	qty := decimal.NewFromInt(a.Throughput - 125)
	qty = qty.Div(decimal.NewFromInt(1024))

	return &engine.LineItem{
		Name:            "Provisioned throughput",
		Unit:            "Mbps",
		UnitMultiplier:  decimal.NewFromFloat(1.0 / 1024.0),
		MonthlyQuantity: decimalPtr(qty),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(a.Region),
			Service:       strPtr("AmazonEC2"),
			ProductFamily: strPtr("Provisioned Throughput"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "volumeApiName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", a.Type))},
				{Key: "usagetype", ValueRegex: strPtr("/VolumeP-Throughput.gp3/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			Unit: strPtr("GiBps-mo"),
		},
	}
}
