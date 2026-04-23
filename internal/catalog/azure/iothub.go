package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

const (
	iotHubFreeSku = "F1"
)

// IoTHub struct represents an IoT Hub
//
// Resource information: https://azure.microsoft.com/en-us/services/iot-hub/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/iot-hub/
type IoTHub struct {
	Address  string
	Region   string
	Sku      string
	Capacity int64
}

func (r *IoTHub) CoreType() string {
	return "IoTHub"
}

func (r *IoTHub) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *IoTHub) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *IoTHub) BuildResource() *engine.Estimate {
	if r.Sku == iotHubFreeSku {
		return &engine.Estimate{
			Name:      r.Address,
			IsSkipped: true,
			NoPrice:   true,
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: r.costComponents(),
	}
}

func (r *IoTHub) costComponents() []*engine.LineItem {
	return r.iotHubCostComponent()
}

func (r *IoTHub) iotHubCostComponent() []*engine.LineItem {
	costComponents := []*engine.LineItem{
		{
			Name:            "IoT Hub",
			Unit:            "units",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(r.Capacity)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("IoT Hub"),
				ProductFamily: strPtr("Internet of Things"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "skuName", Value: strPtr(r.Sku)},
					{Key: "meterName", ValueRegex: regexPtr("Unit$")},
				},
			},
		},
	}

	return costComponents
}

// IoTHubDPS struct represents an IoT Hub DPS
//
// Resource information: https://azure.microsoft.com/en-us/services/iot-hub/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/iot-hub/
type IoTHubDPS struct {
	Address string
	Region  string
	Sku     string

	MonthlyOperations *int64 `c3x_usage:"monthly_operations"`
}

func (r *IoTHubDPS) CoreType() string {
	return "IoTHubDPS"
}

func (r *IoTHubDPS) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_operations", DefaultValue: 0, ValueType: engine.Int64},
	}
}

func (r *IoTHubDPS) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *IoTHubDPS) BuildResource() *engine.Estimate {
	var monthlyOperations *decimal.Decimal

	if r.MonthlyOperations != nil {
		value := decimal.NewFromInt((*r.MonthlyOperations))
		monthlyOperations = decimalPtr(value.Div(decimal.NewFromInt(1000)))
	}

	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            "Device provisioning",
				Unit:            "1k operations",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyOperations,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(r.Region),
					Service:       strPtr("IoT Hub"),
					ProductFamily: strPtr("Internet of Things"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "skuName", Value: strPtr(r.Sku)},
						{Key: "meterName", ValueRegex: regexPtr("Operations$")},
					},
				},
				UsageBased: true,
			},
		},
	}
}
