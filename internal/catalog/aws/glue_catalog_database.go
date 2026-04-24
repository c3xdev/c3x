package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// GlueCatalogDatabase struct represents a serverless AWS Glue catalog. A Glue catalog is a database designed to store
// raw data fetched from Glue crawlers before the data is cleaned and transformed by a Glue job.
//
// GlueCatalogDatabase is just one resource of the wider AWS Glue service, which provides a number of different serverless services
// to build a robust data analytics pipeline.
//
// Resource information: https://aws.amazon.com/glue/
// Pricing information: https://aws.amazon.com/glue/pricing/
type GlueCatalogDatabase struct {
	Address string
	Region  string

	MonthlyObjects  *float64 `c3x_usage:"monthly_objects"`
	MonthlyRequests *float64 `c3x_usage:"monthly_requests"`
}

func (r *GlueCatalogDatabase) CoreType() string {
	return "GlueCatalogDatabase"
}

func (r *GlueCatalogDatabase) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_objects", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_requests", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the GlueCatalogDatabase.
// It uses the `c3x_usage` struct tags to populate data into the GlueCatalogDatabase.
func (r *GlueCatalogDatabase) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid GlueCatalogDatabase struct. GlueCatalogDatabase has the following
// engine.LineItems associated with it:
//
//  1. Storage - charged for every 100,000 objects stored above 1M, per month.
//  2. MonthlyAdditionalRequests - charged per million requests above 1M in a month.
//
// This method is called after the resource is initialised by an IaC provider. See providers folder for more information.
func (r *GlueCatalogDatabase) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			r.storageObjectsCostComponent(),
			r.requestsCostComponent(),
		},
	}
}

func (r *GlueCatalogDatabase) storageObjectsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	var limit float64 = 100000

	if r.MonthlyObjects != nil {
		objects := *r.MonthlyObjects

		if objects < limit {
			objects = 0
		}

		quantity = decimalPtr(decimal.NewFromFloat(objects))
	}

	return &engine.LineItem{
		Name:            "Storage",
		Unit:            "100k objects",
		UnitMultiplier:  decimal.NewFromFloat(limit),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    vendorName,
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSGlue"),
			ProductFamily: strPtr("AWS Glue"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "group", ValueRegex: strPtr("/^data catalog storage$/i")},
			},
		},
		UsageBased: true,
	}
}

func (r *GlueCatalogDatabase) requestsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	var limit float64 = 1000000
	if r.MonthlyRequests != nil {
		requests := *r.MonthlyRequests

		if requests < limit {
			requests = 0
		}

		quantity = decimalPtr(decimal.NewFromFloat(requests))
	}

	return &engine.LineItem{
		Name:            "Requests",
		Unit:            "1M requests",
		UnitMultiplier:  decimal.NewFromFloat(limit),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    vendorName,
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSGlue"),
			ProductFamily: strPtr("AWS Glue"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "group", ValueRegex: strPtr("/^data catalog requests$/i")},
			},
		},
		UsageBased: true,
	}
}
