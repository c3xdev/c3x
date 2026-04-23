package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

var (
	cloudTrailServiceName = strPtr("AWSCloudTrail")

	cloudTrailManagementEvent = "Management events (additional copies)"
	cloudTrailDataEvent       = "Data events"
	cloudTrailInsightEvent    = "Insight events"

	cloudTrailBillingMultiplier = decimal.NewFromInt(100000)
)

// Cloudtrail struct represents a cloudtrail instance to monitor activity across a set of catalog.
// AWS Cloudtrail monitors and records account activity across infrastructure, keeping an audit log of activity.
// This is mostly used for security purposes.
//
// Resource information: https://aws.amazon.com/cloudtrail/
// Pricing information: https://aws.amazon.com/cloudtrail/pricing/
type Cloudtrail struct {
	Address                 string
	Region                  string
	IncludeManagementEvents bool
	IncludeInsightEvents    bool

	MonthlyAdditionalManagementEvents *float64 `c3x_usage:"monthly_additional_management_events"`
	MonthlyDataEvents                 *float64 `c3x_usage:"monthly_data_events"`
	MonthlyInsightEvents              *float64 `c3x_usage:"monthly_insight_events"`
}

func (r *Cloudtrail) CoreType() string {
	return "Cloudtrail"
}

func (r *Cloudtrail) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_additional_management_events", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_data_events", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_insight_events", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the Cloudtrail.
// It uses the `c3x_usage` struct tags to populate data into the Cloudtrail.
func (r *Cloudtrail) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid Cloudtrail struct.
// It returns Cloudtrail as a engine.Estimate with 3 main cost components. All cost components are defined as "events".
// All cost components are charged per 100k events delivered/analyzed.
//
//  1. Additional Management events delivered to S3, charged at $2.00 per 100k management events delivered.
//     Management events are normally priced as free, however if a user specifies an additional replication of events
//     this is charged. We only show this cost therefore if Cloudtrail.IncludeManagementEvents is set. This is set at
//     a per IAC basis.
//  2. Data events delivered to S3, charged at $0.10 per 100k events delivered.
//  3. CloudTrail Insights, charged at $0.35 per 100k events analyzed. This again is configured optionally on a Cloudtrail
//     instance. Hence, we only include the cost component if Cloudtrail.IncludeInsightEvents. This is set at
//     a per IAC basis.
//
// This method is called after the resource is initialised by an IaC provider. See providers folder for more information.
func (r *Cloudtrail) BuildResource() *engine.Estimate {
	var costComponents []*engine.LineItem

	if r.IncludeManagementEvents || r.MonthlyAdditionalManagementEvents != nil {
		costComponents = append(costComponents, r.managementEventCostComponent())
	}

	costComponents = append(costComponents, r.dataEventsCostComponent())

	if r.IncludeInsightEvents || r.MonthlyInsightEvents != nil {
		costComponents = append(costComponents, r.insightEventsCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *Cloudtrail) managementEventCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyAdditionalManagementEvents != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyAdditionalManagementEvents))
	}

	return r.eventCostComponent(cloudTrailManagementEvent, quantity)
}

func (r *Cloudtrail) dataEventsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyDataEvents != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataEvents))
	}

	return r.eventCostComponent(cloudTrailDataEvent, quantity)
}

func (r *Cloudtrail) insightEventsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyInsightEvents != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyInsightEvents))
	}

	return r.eventCostComponent(cloudTrailInsightEvent, quantity)
}

func (r *Cloudtrail) eventCostComponent(name string, quantity *decimal.Decimal) *engine.LineItem {
	productFamily := "Management Tools - AWS CloudTrail Paid Events Recorded"
	if name == cloudTrailDataEvent {
		productFamily = "Management Tools - AWS CloudTrail Data Events Recorded"
	}

	var attrFilters []*engine.AttributeMatch
	if name == cloudTrailInsightEvent {
		productFamily = "Management Tools - AWS CloudTrail Insights Events"
		attrFilters = []*engine.AttributeMatch{
			{Key: "usagetype", ValueRegex: regexPtr(".*-InsightsEvents$")},
		}
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            "100k events",
		UnitMultiplier:  cloudTrailBillingMultiplier,
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:       vendorName,
			Region:           strPtr(r.Region),
			Service:          cloudTrailServiceName,
			ProductFamily:    strPtr(productFamily),
			AttributeFilters: attrFilters,
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
