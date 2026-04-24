package aws

import (
	"time"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

var (
	eksSupportMap = map[string]time.Time{
		"1.29": time.Date(2025, time.March, 23, 0, 0, 0, 0, time.UTC),
		"1.28": time.Date(2024, time.November, 26, 0, 0, 0, 0, time.UTC),
		"1.27": time.Date(2024, time.July, 24, 0, 0, 0, 0, time.UTC),
		"1.26": time.Date(2024, time.June, 11, 0, 0, 0, 0, time.UTC),
		"1.25": time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC),
		"1.24": time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
		"1.23": time.Date(2023, time.October, 11, 0, 0, 0, 0, time.UTC),
	}
)

type EKSCluster struct {
	Address string
	Version string
	Region  string
}

func (r *EKSCluster) CoreType() string {
	return "EKSCluster"
}

func (r *EKSCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EKSCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EKSCluster) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{r.clusterHoursCostComponent()},
		UsageSchema:    r.UsageSchema(),
	}
}

// clusterHoursCostComponent returns the management cost of the EKS cluster. This
// can include extended support cost if the version is not supported by AWS
// anymore. In this case we set a custom price of 0.6$ per hour. This is a
// placeholder until AWS provides the price. and we can look it up in the Pricing
// API.
func (r *EKSCluster) clusterHoursCostComponent() *engine.LineItem {
	baseCost := &engine.LineItem{
		Name:           "EKS cluster",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEKS"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/AmazonEKS-Hours:perCluster/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}

	if r.Version == "" {
		return baseCost
	}

	supportDate := eksSupportMap[r.Version]
	if supportDate.IsZero() {
		return baseCost
	}

	if !supportDate.Before(time.Now()) {
		return baseCost
	}

	baseCost.Name = "EKS cluster (extended support)"
	baseCost.ProductFilter.AttributeFilters = []*engine.AttributeMatch{
		{Key: "usagetype", ValueRegex: strPtr("/AmazonEKS-Hours:extendedSupport/")},
	}

	return baseCost

}
