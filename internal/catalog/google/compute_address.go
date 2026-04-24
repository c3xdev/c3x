package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type ComputeAddress struct {
	Address                string
	Region                 string
	AddressType            string
	Purpose                string
	InstancePurchaseOption string
}

func (r *ComputeAddress) CoreType() string {
	return "ComputeAddress"
}

func (r *ComputeAddress) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ComputeAddress) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeAddress) BuildResource() *engine.Estimate {
	addressType := r.AddressType
	isFreePurpose := r.Purpose != "" && strings.ToLower(r.Purpose) != "gce_endpoint"

	if strings.ToLower(addressType) == "internal" || isFreePurpose {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	costComponents := []*engine.LineItem{}

	switch r.InstancePurchaseOption {
	case "on_demand":
		costComponents = append(costComponents, r.standardVMComputeAddress())
	case "preemptible":
		costComponents = append(costComponents, r.preemptibleVMComputeAddress())
	default:
		costComponents = append(costComponents, r.unusedVMComputeAddress())
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ComputeAddress) standardVMComputeAddress() *engine.LineItem {
	return &engine.LineItem{
		Name:           "IP address (standard VM)",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr("global"),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr("External IP Charge on a Standard VM")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("696"),
		},
	}
}

func (r *ComputeAddress) preemptibleVMComputeAddress() *engine.LineItem {
	return &engine.LineItem{
		Name:           "IP address (preemptible VM)",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr("global"),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr("External IP Charge on a Spot Preemptible VM")},
			},
		},
		PriceFilter: &engine.RateSelector{
			EndUsageAmount: strPtr(""),
		},
	}
}

func (r *ComputeAddress) unusedVMComputeAddress() *engine.LineItem {
	return &engine.LineItem{
		Name:           "IP address (unused)",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("^Static Ip Charge.*")},
			},
		},
		PriceFilter: &engine.RateSelector{
			EndUsageAmount: strPtr(""),
		},
	}
}
