package google

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

var redisNodeTypeNames = map[string]string{
	"REDIS_SHARED_CORE_NANO": "shared core nano",
	"REDIS_STANDARD_SMALL":   "standard small",
	"REDIS_HIGHMEM_MEDIUM":   "highmem medium",
	"REDIS_HIGHMEM_XLARGE":   "highmem xlarge",
}

var redisNodeTypeDescSuffixes = map[string]string{
	"REDIS_SHARED_CORE_NANO": "Shared Core Nano",
	"REDIS_STANDARD_SMALL":   "Standard Small",
	"REDIS_HIGHMEM_MEDIUM":   "Default",
	"REDIS_HIGHMEM_XLARGE":   "Highmem XLarge",
}

type RedisCluster struct {
	Address          string
	Region           string
	NodeType         string
	NodeCount        int
	AOFProvisionedGB int64
	AOFEnabled       bool
	BackupsEnabled   bool
	BackupStorageGB  *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *RedisCluster) CoreType() string {
	return "RedisCluster"
}

func (r *RedisCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *RedisCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *RedisCluster) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.nodeTypeCostComponent())

	if r.AOFEnabled {
		costComponents = append(costComponents, r.aofCostComponent())
	}

	if r.BackupsEnabled {
		costComponents = append(costComponents, r.backupCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *RedisCluster) nodeTypeCostComponent() *engine.LineItem {
	nameSuffix, ok := redisNodeTypeNames[strings.ToUpper(r.NodeType)]
	if !ok {
		nameSuffix = r.NodeType
	}
	name := fmt.Sprintf("Cluster node (%s)", strings.ToLower(nameSuffix))

	descSuffix, ok := redisNodeTypeDescSuffixes[strings.ToUpper(r.NodeType)]
	if !ok {
		descSuffix = r.NodeType
	}
	descriptionRegex := fmt.Sprintf("Redis Cluster Node %s", descSuffix)

	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(r.NodeCount))),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Cloud Memorystore for Redis"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr(descriptionRegex)},
			},
		},
	}
}

func (r *RedisCluster) aofCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "AOF persistence",
		Unit:           "GB",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.AOFProvisionedGB)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Cloud Memorystore for Redis"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("Memorystore for Redis Cluster: AOF Storage")},
			},
		},
	}
}

func (r *RedisCluster) backupCostComponent() *engine.LineItem {
	var backupGB *decimal.Decimal
	if r.BackupStorageGB != nil {
		backupGB = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB).Mul(engine.HourToMonthUnitMultiplier))
	}

	return &engine.LineItem{
		Name:            "Backups",
		Unit:            "GB",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: backupGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Cloud Memorystore for Redis"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr("Memorystore for Redis Cluster: Backups")},
			},
		},
	}
}
