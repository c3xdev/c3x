package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type DMSReplicationInstance struct {
	Address                  string
	Region                   string
	AllocatedStorageGB       int64
	ReplicationInstanceClass string
	MultiAZ                  bool
}

func (r *DMSReplicationInstance) CoreType() string {
	return "DMSReplicationInstance"
}

func (r *DMSReplicationInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *DMSReplicationInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DMSReplicationInstance) BuildResource() *engine.Estimate {
	instanceTypeParts := strings.Split(r.ReplicationInstanceClass, ".")
	if len(instanceTypeParts) < 3 {
		return &engine.Estimate{
			Name:      r.Address,
			NoPrice:   true,
			IsSkipped: true,
		}
	}
	instanceType := strings.Join(instanceTypeParts[1:], ".")
	instanceFamily := instanceTypeParts[1]

	costComponents := make([]*engine.LineItem, 0)
	costComponents = append(costComponents, r.instanceCostComponent(instanceType))
	costComponents = append(costComponents, r.storageCostComponent(instanceFamily))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *DMSReplicationInstance) instanceCostComponent(instanceType string) *engine.LineItem {
	availabilityZone := "Single"
	if r.MultiAZ {
		availabilityZone = "Multiple"
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("Instance (%s)", instanceType),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AWSDatabaseMigrationSvc"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "instanceType", Value: strPtr(instanceType)},
				{Key: "availabilityZone", Value: strPtr(availabilityZone)},
			},
		},
	}
}

func (r *DMSReplicationInstance) storageCostComponent(instanceFamily string) *engine.LineItem {
	availabilityZone := "Single"
	if r.MultiAZ {
		availabilityZone = "Multiple"
	}

	baseStorageSize := r.AllocatedStorageGB
	var freeStorageSize int64
	switch instanceFamily {
	case "c4":
		freeStorageSize = 100
	case "r4":
		freeStorageSize = 100
	case "r5":
		freeStorageSize = 100
	case "t2":
		freeStorageSize = 50
	case "t3":
		freeStorageSize = 50
	}
	var storageSize int64
	if baseStorageSize > freeStorageSize {
		storageSize = baseStorageSize - freeStorageSize
	}

	return &engine.LineItem{
		Name:            "Storage (general purpose SSD, gp2)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(storageSize)),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AWSDatabaseMigrationSvc"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "storageMedia", Value: strPtr("SSD")},
				{Key: "availabilityZone", Value: strPtr(availabilityZone)},
			},
		},
	}
}
