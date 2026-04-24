package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// PrivateDnsResolverOutboundEndpoint struct represents a Azure DNS Private Resolver Outbound Endpoint.
//
// Resource information: https://learn.microsoft.com/en-us/azure/dns/dns-private-resolver-overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/dns/
type PrivateDnsResolverOutboundEndpoint struct {
	Address string
	Region  string
}

// CoreType returns the name of this resource type
func (r *PrivateDnsResolverOutboundEndpoint) CoreType() string {
	return "PrivateDnsResolverOutboundEndpoint"
}

// UsageSchema defines a list which represents the usage schema of PrivateDnsResolverOutboundEndpoint.
func (r *PrivateDnsResolverOutboundEndpoint) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the PrivateDnsResolverOutboundEndpoint.
// It uses the `c3x_usage` struct tags to populate data into the PrivateDnsResolverOutboundEndpoint.
func (r *PrivateDnsResolverOutboundEndpoint) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid PrivateDnsResolverOutboundEndpoint struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *PrivateDnsResolverOutboundEndpoint) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            "Outbound endpoint",
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
						{Key: "meterName", ValueRegex: regexPtr("Private Resolver Outbound Endpoint")},
					},
				},
			},
		},
	}
}
