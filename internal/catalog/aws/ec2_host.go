package aws

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// EC2Host defines an AWS EC2 dedicated host. It supports multiple instance families & allows
// you to run workloads on a physical server dedicated for your use. You can use on-demand or
// reservation pricing.
//
// See more resource information here: https://aws.amazon.com/ec2/dedicated-hosts/
//
// See the pricing information here: https://aws.amazon.com/ec2/dedicated-hosts/pricing/
type EC2Host struct {
	Address                       string
	Region                        string
	InstanceType                  string
	InstanceFamily                string
	ReservedInstanceTerm          *string `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption *string `c3x_usage:"reserved_instance_payment_option"`
}

func (r *EC2Host) CoreType() string {
	return "EC2Host"
}

func (r *EC2Host) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "reserved_instance_term", DefaultValue: "", ValueType: engine.String},
		{Key: "reserved_instance_payment_option", DefaultValue: "", ValueType: engine.String},
	}
}

func (r *EC2Host) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EC2Host) BuildResource() *engine.Estimate {
	purchaseOptionLabel := "on-demand"
	priceFilter := &engine.RateSelector{
		PurchaseOption: strPtr("on_demand"),
	}

	var err error
	if r.ReservedInstanceTerm != nil {
		resolver := &ec2HostReservationResolver{
			term:          strVal(r.ReservedInstanceTerm),
			paymentOption: strVal(r.ReservedInstancePaymentOption),
		}

		priceFilter, err = resolver.PriceFilter()

		if err != nil {
			logging.Logger.Warn().Msg(err.Error())
		}

		purchaseOptionLabel = "reserved"
	}

	instanceFamily := r.InstanceFamily

	if r.InstanceType != "" {
		split := strings.Split(r.InstanceType, ".")
		if len(split) > 0 {
			instanceFamily = split[0]
		}
	}

	hostPurchaseType := "HostUsage"

	if purchaseOptionLabel == "reserved" {
		hostPurchaseType = "ReservedHostUsage"
	}

	hostAttributeFilters := []*engine.AttributeMatch{
		{Key: "usagetype", ValueRegex: regexPtr(fmt.Sprintf("%s:%s$", hostPurchaseType, instanceFamily))},
	}

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("EC2 Dedicated Host (%s, %s)", purchaseOptionLabel, instanceFamily),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName:       strPtr("aws"),
				Region:           strPtr(r.Region),
				Service:          strPtr("AmazonEC2"),
				ProductFamily:    strPtr("Dedicated Host"),
				AttributeFilters: hostAttributeFilters,
			},
			PriceFilter: priceFilter,
		},
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

type ec2HostReservationResolver struct {
	term          string
	paymentOption string
}

// PriceFilter implementation for ec2HostReservationResolver
// Allowed values for ReservedInstanceTerm: ["1_year", "3_year"]
// Allowed values for ReservedInstancePaymentOption: ["all_upfront", "partial_upfront", "no_upfront"]
func (r ec2HostReservationResolver) PriceFilter() (*engine.RateSelector, error) {
	purchaseOptionLabel := "reserved"
	def := &engine.RateSelector{
		PurchaseOption: strPtr(purchaseOptionLabel),
	}
	termLength := reservedTermsMapping[r.term]
	purchaseOption := reservedHostPaymentOptionMapping[r.paymentOption]
	validTerms := sliceOfKeysFromMap(reservedTermsMapping)
	if !stringInSlice(validTerms, r.term) {
		return def, fmt.Errorf("Invalid reserved_instance_term, ignoring reserved options. Expected: %s. Got: %s", strings.Join(validTerms, ", "), r.term)
	}
	validOptions := sliceOfKeysFromMap(reservedPaymentOptionMapping)

	if !stringInSlice(validOptions, r.paymentOption) {
		return def, fmt.Errorf("Invalid reserved_instance_payment_option, ignoring reserved options. Expected: %s. Got: %s", strings.Join(validOptions, ", "), r.paymentOption)
	}
	return &engine.RateSelector{
		PurchaseOption:     strPtr(purchaseOptionLabel),
		StartUsageAmount:   strPtr("0"),
		TermLength:         strPtr(termLength),
		TermPurchaseOption: strPtr(purchaseOption),
	}, nil
}
