package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type RedisInstance struct {
	Address      string
	Region       string
	Tier         string
	MemorySizeGB float64
}

func (r *RedisInstance) CoreType() string {
	return "RedisInstance"
}

func (r *RedisInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *RedisInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *RedisInstance) BuildResource() *engine.Estimate {
	serviceTier := "Basic"

	var tierMapping = map[string]string{
		"BASIC":       "Basic",
		"STANDARD_HA": "Standard",
	}

	if r.Tier != "" {
		serviceTier = tierMapping[r.Tier]
	}

	var memorySize = r.MemorySizeGB
	var capacityTier string

	if memorySize >= 1 && memorySize <= 4 {
		capacityTier = "M1"
	} else if memorySize >= 5 && memorySize <= 10 {
		capacityTier = "M2"
	} else if memorySize >= 11 && memorySize <= 35 {
		capacityTier = "M3"
	} else if memorySize >= 36 && memorySize <= 100 {
		capacityTier = "M4"
	} else {
		capacityTier = "M5"
	}

	description := fmt.Sprintf("/Redis Capacity %s %s/", serviceTier, capacityTier)
	name := fmt.Sprintf("Redis instance (%s, %s)", strings.ToLower(serviceTier), capacityTier)

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           name,
				Unit:           "GB",
				UnitMultiplier: engine.HourToMonthUnitMultiplier,
				HourlyQuantity: decimalPtr(decimal.NewFromFloat(memorySize)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Cloud Memorystore for Redis"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", ValueRegex: strPtr(description)},
					},
				},
			},
		}, UsageSchema: r.UsageSchema(),
	}
}
