package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type ACMPCACertificateAuthority struct {
	Address         string
	Region          string
	UsageMode       string
	MonthlyRequests *int64 `c3x_usage:"monthly_requests"`
}

func (r *ACMPCACertificateAuthority) CoreType() string {
	return "ACMPCACertificateAuthority"
}

func (r *ACMPCACertificateAuthority) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *ACMPCACertificateAuthority) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ACMPCACertificateAuthority) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.certificateAuthorityCostComponent(),
	}

	if r.MonthlyRequests != nil {
		monthlyCertificatesRequests := decimal.NewFromInt(*r.MonthlyRequests)

		if r.shortLived() {
			costComponents = append(costComponents, r.certificateCostComponent("Certificates (short-lived)", "0", &monthlyCertificatesRequests))
		} else {
			certificateTierLimits := []int{1000, 9000}
			certificateTiers := usage.CalculateTierBuckets(monthlyCertificatesRequests, certificateTierLimits)

			if certificateTiers[0].GreaterThan(decimal.NewFromInt(0)) {
				costComponents = append(costComponents, r.certificateCostComponent("Certificates (first 1K)", "0", &certificateTiers[0]))
			}

			if certificateTiers[1].GreaterThan(decimal.NewFromInt(0)) {
				costComponents = append(costComponents, r.certificateCostComponent("Certificates (next 9K)", "1000", &certificateTiers[1]))
			}

			if certificateTiers[2].GreaterThan(decimal.NewFromInt(0)) {
				costComponents = append(costComponents, r.certificateCostComponent("Certificates (over 10K)", "10000", &certificateTiers[2]))
			}
		}
	} else {
		var unknown *decimal.Decimal
		if r.shortLived() {
			costComponents = append(costComponents, r.certificateCostComponent("Certificates (short-lived)", "0", unknown))
		} else {
			costComponents = append(costComponents, r.certificateCostComponent("Certificates (first 1K)", "0", unknown))
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ACMPCACertificateAuthority) shortLived() bool {
	return strings.ToLower(r.UsageMode) == "short_lived_certificate"
}

func (r *ACMPCACertificateAuthority) certificateAuthorityCostComponent() *engine.LineItem {
	name := "Private certificate authority"
	regex := "/-PaidPrivateCA/"
	if r.shortLived() {
		name = "Private certificate authority (short-lived certificate mode)"
		regex = "/-ShortLivedCertificatePrivateCA/"
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSCertificateManager"),
			ProductFamily: strPtr("AWS Certificate Manager"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: &regex},
			},
		},
	}
}

func (r *ACMPCACertificateAuthority) certificateCostComponent(displayName string, usageTier string, monthlyQuantity *decimal.Decimal) *engine.LineItem {
	regex := "/-PrivateCertificatesIssued/"
	if r.shortLived() {
		regex = "/-ShortLivedCertificatesIssued/"
	}

	return &engine.LineItem{
		Name:            displayName,
		Unit:            "requests",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSCertificateManager"),
			ProductFamily: strPtr("AWS Certificate Manager"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: &regex},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}
