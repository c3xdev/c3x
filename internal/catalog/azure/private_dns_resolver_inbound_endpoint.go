package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// PrivateDnsResolverInboundEndpoint struct represents a Azure DNS Private Resolver Inbound Endpoint.
//
// Resource information: https://learn.microsoft.com/en-us/azure/dns/dns-private-resolver-overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/dns/
type PrivateDnsResolverInboundEndpoint struct {
	Address string
	Region  string
}

// CoreType returns the name of this resource type
func (r *PrivateDnsResolverInboundEndpoint) CoreType() string {
	return "PrivateDnsResolverInboundEndpoint"
}

// UsageSchema defines a list which represents the usage schema of PrivateDnsResolverInboundEndpoint.
func (r *PrivateDnsResolverInboundEndpoint) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the PrivateDnsResolverInboundEndpoint.
// It uses the `c3x_usage` struct tags to populate data into the PrivateDnsResolverInboundEndpoint.
func (r *PrivateDnsResolverInboundEndpoint) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid PrivateDnsResolverInboundEndpoint struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *PrivateDnsResolverInboundEndpoint) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            "Inbound endpoint",
				Unit:            "months",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(dnsZoneRegion(r.Region)),
					Service:       strPtr("Azure DNS"),
					ProductFamily: strPtr("Networking"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "skuName", ValueRegex: regexPtr("Private Resolver")},
						{Key: "meterName", ValueRegex: regexPtr("Private Resolver Inbound Endpoint")},
					},
				},
			},
		},
	}
}
