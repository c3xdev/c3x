package azure

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type WindowsVirtualMachine struct {
	Address                               string
	Region                                string
	Size                                  string
	LicenseType                           string
	AdditionalCapabilitiesUltraSSDEnabled bool
	OSDiskData                            *ManagedDiskData
	MonthlyHours                          *float64     `c3x_usage:"monthly_hrs"`
	OSDisk                                *OSDiskUsage `c3x_usage:"os_disk"`
	IsDevTest                             bool
}

type OSDiskUsage struct {
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

var OSDiskUsageSchema = []*engine.ConsumptionField{
	{ValueType: engine.Int64, DefaultValue: 0, Key: "monthly_disk_operations"},
}

func (r *WindowsVirtualMachine) CoreType() string {
	return "WindowsVirtualMachine"
}

func (r *WindowsVirtualMachine) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", ValueType: engine.Float64, DefaultValue: 0},
		{
			Key:          "os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "os_disk", Items: OSDiskUsageSchema},
		},
	}
}

func (r *WindowsVirtualMachine) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *WindowsVirtualMachine) BuildResource() *engine.Estimate {
	region := r.Region

	instanceType := r.Size
	licenseType := r.LicenseType

	costComponents := []*engine.LineItem{windowsVirtualMachineCostComponent(region, instanceType, licenseType, r.MonthlyHours, r.IsDevTest)}

	if r.AdditionalCapabilitiesUltraSSDEnabled {
		costComponents = append(costComponents, ultraSSDReservationCostComponent(region))
	}

	subResources := make([]*engine.Estimate, 0)

	var monthlyDiskOperations *decimal.Decimal
	if r.OSDisk != nil && r.OSDisk.MonthlyDiskOperations != nil {
		monthlyDiskOperations = decimalPtr(decimal.NewFromInt(*r.OSDisk.MonthlyDiskOperations))
	}
	if r.OSDiskData != nil {
		osDisk := osDiskSubResource(region, r.OSDiskData.DiskType, r.OSDiskData.DiskSizeGB, r.OSDiskData.DiskIOPSReadWrite, r.OSDiskData.DiskMBPSReadWrite, monthlyDiskOperations)
		if osDisk != nil {
			subResources = append(subResources, osDisk)
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}
}

func windowsVirtualMachineCostComponent(region string, instanceType string, licenseType string, monthlyHours *float64, isDevTest bool) *engine.LineItem {
	purchaseOption := "Consumption"
	purchaseOptionLabel := "pay as you go"

	productNameRe := "/(Series )?Windows$/i"
	if strings.HasPrefix(instanceType, "Basic_") {
		productNameRe = "/Basic Windows$/"
	} else if !strings.HasPrefix(instanceType, "Standard_") {
		instanceType = fmt.Sprintf("Standard_%s", instanceType)
	}

	if strings.ToLower(licenseType) == "windows_client" || strings.ToLower(licenseType) == "windows_server" {
		purchaseOption = "DevTestConsumption"
		purchaseOptionLabel = "hybrid benefit"
	}

	if isDevTest {
		purchaseOption = "DevTestConsumption"
		purchaseOptionLabel = "dev/test"
	}

	qty := engine.HourToMonthUnitMultiplier
	if monthlyHours != nil {
		qty = decimal.NewFromFloat(*monthlyHours)
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("Instance usage (Windows, %s, %s)", purchaseOptionLabel, instanceType),
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(qty),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Virtual Machines"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
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
