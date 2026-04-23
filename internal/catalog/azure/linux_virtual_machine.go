package azure

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type LinuxVirtualMachine struct {
	Address         string
	Region          string
	Size            string
	UltraSSDEnabled bool
	OSDiskData      *ManagedDiskData
	OSDisk          *OSDiskUsage `c3x_usage:"os_disk"`
	MonthlyHrs      *float64     `c3x_usage:"monthly_hrs"`
}

func (r *LinuxVirtualMachine) CoreType() string {
	return "LinuxVirtualMachine"
}

func (r *LinuxVirtualMachine) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", ValueType: engine.Float64, DefaultValue: 0},
		{
			Key:          "os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "os_disk", Items: OSDiskUsageSchema},
		},
	}
}

func (r *LinuxVirtualMachine) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *LinuxVirtualMachine) BuildResource() *engine.Estimate {
	instanceType := r.Size

	costComponents := []*engine.LineItem{linuxVirtualMachineCostComponent(r.Region, instanceType, r.MonthlyHrs)}

	if r.UltraSSDEnabled {
		costComponents = append(costComponents, ultraSSDReservationCostComponent(r.Region))
	}

	subResources := make([]*engine.Estimate, 0)

	var monthlyDiskOperations *decimal.Decimal
	if r.OSDisk != nil && r.OSDisk.MonthlyDiskOperations != nil {
		monthlyDiskOperations = decimalPtr(decimal.NewFromInt(*r.OSDisk.MonthlyDiskOperations))
	}

	if r.OSDiskData != nil {
		osDisk := osDiskSubResource(r.Region, r.OSDiskData.DiskType, r.OSDiskData.DiskSizeGB, r.OSDiskData.DiskIOPSReadWrite, r.OSDiskData.DiskMBPSReadWrite, monthlyDiskOperations)
		subResources = append(subResources, osDisk)
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}
}

func linuxVirtualMachineCostComponent(region string, instanceType string, monthlyHours *float64) *engine.LineItem {
	purchaseOption := "Consumption"
	purchaseOptionLabel := "pay as you go"

	productNameRe := "/Series( Linux)?$/i"
	if strings.HasPrefix(strings.ToLower(instanceType), "basic_") {
		productNameRe = "/Series Basic$/"
	} else if !strings.HasPrefix(strings.ToLower(instanceType), "standard_") {
		instanceType = fmt.Sprintf("Standard_%s", instanceType)
	}

	qty := engine.HourToMonthUnitMultiplier
	if monthlyHours != nil {
		qty = decimal.NewFromFloat(*monthlyHours)
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("Instance usage (Linux, %s, %s)", purchaseOptionLabel, instanceType),
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(qty),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Virtual Machines"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: strPtr("/^(?!.*(Expired|Free)$).*$/i")},
				{Key: "skuName", ValueRegex: strPtr("/^(?!.*(Low Priority|Spot)$).*$/i")},
				{Key: "armSkuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", instanceType))},
				{Key: "productName", ValueRegex: strPtr(productNameRe)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr(purchaseOption),
			Unit:           strPtr("1 Hour"),
		},
	}
}
