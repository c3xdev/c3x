package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// SearchDomain struct represents a managed search domain for both Amazon OpenSearch Service
// (formerly known as Amazon Elasticsearch Service) and AWS Elasticsearch Domain. These services
// provide fully managed search and analytics engines based on Elasticsearch and OpenSearch,
// respectively, allowing you to easily deploy, secure, and scale search solutions.
//
// Amazon OpenSearch Service and AWS Elasticsearch Domain offer a variety of instance types and
// configurations to accommodate different workloads, including dedicated master nodes, warm nodes,
// and EBS storage options. The pricing depends on the chosen instance types, instance counts,
// storage options, and additional features.
//
// For more information about Amazon OpenSearch Service and AWS Elasticsearch Domain functionality
// and pricing, refer to the following links:
//
// Amazon OpenSearch Service:
// Resource information: https://aws.amazon.com/opensearch-service/
// Pricing information: https://aws.amazon.com/opensearch-service/pricing/
//
// AWS Elasticsearch Domain:
// Resource information: https://aws.amazon.com/elasticsearch-service/
// Pricing information: https://aws.amazon.com/elasticsearch-service/pricing/
type SearchDomain struct {
	Address              string
	Region               string
	ClusterInstanceType  string
	ClusterInstanceCount *int64 // If this is nil it will default to 1

	EBSEnabled    bool
	EBSVolumeType string
	EBSVolumeSize *float64 // if this is nil it will default to 8
	EBSIOPS       *float64 // if this is nil it will default to 1
	EBSThroughput *float64 // if this is nil it will default to 0

	ClusterDedicatedMasterEnabled bool
	ClusterDedicatedMasterType    string
	ClusterDedicatedMasterCount   *int64 // if this is nil it will default to 3

	ClusterWarmEnabled bool
	ClusterWarmType    string
	ClusterWarmCount   *int64
}

func (r *SearchDomain) CoreType() string {
	return "SearchDomain"
}

func (r *SearchDomain) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *SearchDomain) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SearchDomain) BuildResource() *engine.Estimate {
	defaultClusterInstanceType := "m4.large.elasticsearch"

	instanceType := defaultClusterInstanceType
	if r.ClusterInstanceType != "" {
		instanceType = r.ClusterInstanceType
	}

	instanceCount := int64(1)
	if r.ClusterInstanceCount != nil {
		instanceCount = *r.ClusterInstanceCount
	}

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("Instance (on-demand, %s)", instanceType),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(instanceCount)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Amazon OpenSearch Service Instance"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
					{Key: "instanceType", Value: r.opensearchifyClusterInstanceType(instanceType)},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
		},
	}

	if r.EBSEnabled {
		gbVal := decimal.NewFromFloat(float64(defaultVolumeSize))
		if r.EBSVolumeSize != nil {
			gbVal = decimal.NewFromFloat(*r.EBSVolumeSize)
		}

		ebsType := "gp2"
		if r.EBSVolumeType != "" {
			ebsType = r.EBSVolumeType
		}

		ebsTypeMap := map[string]string{
			"gp3":      "GP3",
			"gp2":      "GP2",
			"io1":      "PIOPS-Storage",
			"standard": "Magnetic",
		}

		ebsFilter := "gp2"
		if val, ok := ebsTypeMap[ebsType]; ok {
			ebsFilter = val
		}

		costComponents = append(costComponents, &engine.LineItem{
			Name:            fmt.Sprintf("Storage (%s)", ebsType),
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: &gbVal,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Amazon OpenSearch Service Volume"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/ES.+-Storage/")},
					{Key: "storageMedia", Value: strPtr(ebsFilter)},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
		})

		if strings.ToLower(ebsType) == "gp3" {
			paidThroughput := decimal.NewFromInt(0)
			if r.EBSThroughput != nil {
				throughput := decimal.NewFromFloat(*r.EBSThroughput)
				paidThroughput = r.calculateBillableThroughput(throughput, gbVal)
				paidThroughput = paidThroughput.Mul(decimal.NewFromInt(instanceCount))
			}

			costComponents = append(costComponents, &engine.LineItem{
				Name:            fmt.Sprintf("Throughput (%s)", ebsType),
				Unit:            "Mbps",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: &paidThroughput,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonES"),
					ProductFamily: strPtr("Amazon OpenSearch Service Volume"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "storageMedia", Value: strPtr("GP3-ThroughPut-Storage")},
					},
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("on_demand"),
				},
			})
		}

		if strings.ToLower(ebsType) == "io1" {
			iopsVal := decimal.NewFromInt(1)
			if r.EBSIOPS != nil {
				iopsVal = decimal.NewFromFloat(*r.EBSIOPS)

				if iopsVal.LessThan(decimal.NewFromInt(1)) {
					iopsVal = decimal.NewFromInt(1)
				}
			}

			costComponents = append(costComponents, &engine.LineItem{
				Name:            fmt.Sprintf("Storage IOPS (%s)", ebsType),
				Unit:            "IOPS",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: &iopsVal,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonES"),
					ProductFamily: strPtr("Amazon OpenSearch Service Volume"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ES:PIOPS/")},
						{Key: "storageMedia", Value: strPtr("PIOPS")},
					},
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("on_demand"),
				},
			})
		}
	}

	if r.ClusterDedicatedMasterEnabled {
		dedicatedMasterType := defaultClusterInstanceType
		if r.ClusterDedicatedMasterType != "" {
			dedicatedMasterType = r.ClusterDedicatedMasterType
		}

		dedicatedMasterCount := int64(3)
		if r.ClusterDedicatedMasterCount != nil {
			dedicatedMasterCount = *r.ClusterDedicatedMasterCount
		}

		costComponents = append(costComponents, &engine.LineItem{
			Name:           fmt.Sprintf("Dedicated master (on-demand, %s)", dedicatedMasterType),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(dedicatedMasterCount)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Amazon OpenSearch Service Instance"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
					{Key: "instanceType", Value: r.opensearchifyClusterInstanceType(dedicatedMasterType)},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
		})
	}

	if r.ClusterWarmEnabled && r.ClusterWarmType != "" {
		clusterWarmCount := int64(0)
		if r.ClusterWarmCount != nil {
			clusterWarmCount = *r.ClusterWarmCount
		}

		if clusterWarmCount > 0 {
			costComponents = append(costComponents, &engine.LineItem{
				Name:           fmt.Sprintf("UltraWarm instance (on-demand, %s)", r.ClusterWarmType),
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(clusterWarmCount)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonES"),
					ProductFamily: strPtr("Amazon OpenSearch Service Instance"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
						{Key: "instanceType", Value: r.opensearchifyClusterInstanceType(r.ClusterWarmType)},
					},
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("on_demand"),
				},
			})
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *SearchDomain) opensearchifyClusterInstanceType(instanceType string) *string {
	s := strings.Replace(instanceType, ".elasticsearch", ".search", 1)
	return &s
}

// calculateBillableThroughput calculates billable throughput
// - 125 MiB/s free for volumes up to 170 GB
// - +250 MiB/s free for every 3 TB for volumes above 170 GB (storage is rounded up).
func (r *SearchDomain) calculateBillableThroughput(throughput decimal.Decimal, storage decimal.Decimal) decimal.Decimal {
	if storage.LessThanOrEqual(decimal.NewFromInt(170)) {
		if throughput.GreaterThan(decimal.NewFromInt(125)) {
			return throughput.Sub(decimal.NewFromInt(125))
		}

		return decimal.Zero
	}

	maxThroughput := decimal.NewFromInt(1000)

	incFactor := storage.Div(decimal.NewFromInt(3000)).RoundUp(0)
	freeThroughput := decimal.Min(maxThroughput, incFactor.Mul(decimal.NewFromInt(250)))

	if throughput.GreaterThan(freeThroughput) {
		return throughput.Sub(freeThroughput)
	}

	return decimal.Zero
}
