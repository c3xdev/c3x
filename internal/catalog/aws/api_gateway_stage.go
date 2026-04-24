package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type APIGatewayStage struct {
	Address          string
	Region           string
	CacheClusterSize float64
	CacheEnabled     bool
}

func (r *APIGatewayStage) CoreType() string {
	return "APIGatewayStage"
}

func (r *APIGatewayStage) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *APIGatewayStage) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *APIGatewayStage) BuildResource() *engine.Estimate {
	if !r.CacheEnabled {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	region := r.Region

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           fmt.Sprintf("Cache memory (%s GB)", decimal.NewFromFloat(r.CacheClusterSize)),
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(region),
					Service:       strPtr("AmazonApiGateway"),
					ProductFamily: strPtr("Amazon API Gateway Cache"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "cacheMemorySizeGb", ValueRegex: strPtr(fmt.Sprintf("/%s/", decimal.NewFromFloat(r.CacheClusterSize)))},
					},
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
