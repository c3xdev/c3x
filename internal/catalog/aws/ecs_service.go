package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type ECSService struct {
	Address                        string
	LaunchType                     string
	Region                         string
	DesiredCount                   int64
	MemoryGB                       float64
	VCPU                           float64
	InferenceAcceleratorDeviceType string
}

func (r *ECSService) CoreType() string {
	return "ECSService"
}

func (r *ECSService) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ECSService) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ECSService) BuildResource() *engine.Estimate {
	if r.LaunchType != "FARGATE" {
		return &engine.Estimate{
			Name:        r.Address,
			IsSkipped:   true,
			NoPrice:     true,
			UsageSchema: r.UsageSchema(),
		}
	}

	costComponents := []*engine.LineItem{
		{
			Name:           "Per GB per hour",
			Unit:           "GB",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: decimalPtr(decimal.NewFromFloat(r.MemoryGB * float64(r.DesiredCount))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonECS"),
				ProductFamily: strPtr("Compute"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/Fargate-GB-Hours/")},
				},
			},
		},
		{
			Name:           "Per vCPU per hour",
			Unit:           "CPU",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: decimalPtr(decimal.NewFromFloat(r.VCPU * float64(r.DesiredCount))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonECS"),
				ProductFamily: strPtr("Compute"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/Fargate-vCPU-Hours:perCPU/")},
				},
			},
		},
	}

	if r.InferenceAcceleratorDeviceType != "" {
		costComponents = append(costComponents, &engine.LineItem{
			Name:           fmt.Sprintf("Inference accelerator (%s)", r.InferenceAcceleratorDeviceType),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(r.DesiredCount)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonEI"),
				ProductFamily: strPtr("Elastic Inference"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", r.InferenceAcceleratorDeviceType))},
				},
			},
		})
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
