package aws

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	eng "github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type ElastiCacheCluster struct {
	Address                       string
	Region                        string
	HasReplicationGroup           bool
	NodeType                      string
	Engine                        string
	CacheNodes                    int64
	SnapshotRetentionLimit        int64
	SnapshotStorageSizeGB         *float64 `c3x_usage:"snapshot_storage_size_gb"`
	ReservedInstanceTerm          *string  `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption *string  `c3x_usage:"reserved_instance_payment_option"`
}

func (r *ElastiCacheCluster) CoreType() string {
	return "ElastiCacheCluster"
}

func (r *ElastiCacheCluster) UsageSchema() []*eng.ConsumptionField {
	return []*eng.ConsumptionField{
		{Key: "snapshot_storage_size_gb", ValueType: eng.Float64, DefaultValue: 0},
		{Key: "reserved_instance_term", DefaultValue: "", ValueType: eng.String},
		{Key: "reserved_instance_payment_option", DefaultValue: "", ValueType: eng.String},
	}
}

func (r *ElastiCacheCluster) PopulateUsage(u *eng.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ElastiCacheCluster) BuildResource() *eng.Estimate {
	if r.HasReplicationGroup {
		return &eng.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	costComponents := []*eng.LineItem{
		r.elastiCacheCostComponent(false),
	}

	if strings.ToLower(r.Engine) == "redis" && r.SnapshotRetentionLimit > 1 {
		costComponents = append(costComponents, r.backupStorageCostComponent())
	}

	return &eng.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
	}
}

func (r *ElastiCacheCluster) elastiCacheCostComponent(autoscaling bool) *eng.LineItem {
	purchaseOptionLabel := "on-demand"
	priceFilter := &eng.RateSelector{
		PurchaseOption: strPtr("on_demand"),
	}
	if r.ReservedInstanceTerm != nil {
		resolver := &elasticacheReservationResolver{
			term:          strVal(r.ReservedInstanceTerm),
			paymentOption: strVal(r.ReservedInstancePaymentOption),
			cacheNodeType: r.NodeType,
		}
		reservedFilter, err := resolver.PriceFilter()
		if err != nil {
			logging.Logger.Warn().Msg(err.Error())
		} else {
			priceFilter = reservedFilter
		}
		purchaseOptionLabel = "reserved"
	}

	nameParams := []string{purchaseOptionLabel, r.NodeType}
	if autoscaling {
		nameParams = append(nameParams, "autoscaling")
	}

	return &eng.LineItem{
		Name:           fmt.Sprintf("ElastiCache (%s)", strings.Join(nameParams, ", ")),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.CacheNodes)),
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonElastiCache"),
			ProductFamily: strPtr("Cache Instance"),
			AttributeFilters: []*eng.AttributeMatch{
				{Key: "instanceType", Value: strPtr(r.NodeType)},
				{Key: "locationType", Value: strPtr("AWS Region")},
				{Key: "cacheEngine", Value: strPtr(cases.Title(language.English).String(r.Engine))},
				{Key: "currentGeneration", Value: strPtr("Yes")},
			},
		},
		PriceFilter: priceFilter,
		UsageBased:  autoscaling,
	}
}

func (r *ElastiCacheCluster) backupStorageCostComponent() *eng.LineItem {
	var monthlyBackupStorageGB *decimal.Decimal

	backupRetention := r.SnapshotRetentionLimit - 1

	if r.SnapshotStorageSizeGB != nil {
		snapshotSize := decimal.NewFromFloat(*r.SnapshotStorageSizeGB)
		monthlyBackupStorageGB = decimalPtr(snapshotSize.Mul(decimal.NewFromInt(backupRetention)))
	}

	return &eng.LineItem{
		Name:            "Backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyBackupStorageGB,
		ProductFilter: &eng.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonElastiCache"),
			ProductFamily: strPtr("Storage Snapshot"),
		},
		UsageBased: true,
	}
}

type elasticacheReservationResolver struct {
	term          string
	paymentOption string
	cacheNodeType string
}

func (r elasticacheReservationResolver) isElasticacheReservedNodeLegacyOffering() bool {
	for k := range elasticacheReservedNodeCacheLegacyOfferings {
		if k == r.paymentOption {
			return true
		}
	}
	return false
}

// PriceFilter implementation for elasticacheReservationResolver
// Allowed values for ReservedInstanceTerm: ["1_year", "3_year"]
// Allowed values for ReservedInstancePaymentOption: ["all_upfront", "partial_upfront", "no_upfront"] for non legacy reservation nodes
// Allowed values for ReservedInstancePaymentOption: ["heavy_utilization", "medium_utilization", "light_utilization"] for legacy reservation nodes
// Legacy reservation nodes: "t2", "m3", "m4", "r3", "r4". (See elasticacheReservedNodeLegacyTypes in util.go)
// Corner Case: In the case of legacy reservation cache nodes unfortunately, for a specified node type, the allowed ReservedInstancePaymentOption may differ in different regions.
//
//	Because of this, in the case of a legacy reservation, a warning is raised to the user.
func (r elasticacheReservationResolver) PriceFilter() (*eng.RateSelector, error) {
	purchaseOptionLabel := "reserved"
	def := &eng.RateSelector{
		PurchaseOption: strPtr(purchaseOptionLabel),
	}
	termLength := reservedTermsMapping[r.term]
	var purchaseOption string
	if r.isElasticacheReservedNodeLegacyOffering() {
		purchaseOption = elasticacheReservedNodeCacheLegacyOfferings[r.paymentOption]
	} else {
		purchaseOption = reservedPaymentOptionMapping[r.paymentOption]
	}
	validTerms := sliceOfKeysFromMap(reservedTermsMapping)
	if !stringInSlice(validTerms, r.term) {
		return def, fmt.Errorf("Invalid reserved_instance_term, ignoring reserved options. Expected: %s. Got: %s", strings.Join(validTerms, ", "), r.term)
	}
	validOptions := append(sliceOfKeysFromMap(reservedPaymentOptionMapping), sliceOfKeysFromMap(elasticacheReservedNodeCacheLegacyOfferings)...)

	if !stringInSlice(validOptions, r.paymentOption) {
		return def, fmt.Errorf("Invalid reserved_instance_payment_option, ignoring reserved options. Expected: %s. Got: %s", strings.Join(validOptions, ", "), r.paymentOption)
	}
	nodeType := strings.Split(r.cacheNodeType, ".")[1] // Get node type from cache node type. cache.m3.large -> m3
	if stringInSlice(elasticacheReservedNodeLegacyTypes, nodeType) {
		logging.Logger.Warn().Msgf("No products found is possible for legacy nodes %s if provided payment option is not supported by the region.", strings.Join(elasticacheReservedNodeLegacyTypes, ", "))
	}
	return &eng.RateSelector{
		PurchaseOption:     strPtr(purchaseOptionLabel),
		StartUsageAmount:   strPtr("0"),
		TermLength:         strPtr(termLength),
		TermPurchaseOption: strPtr(purchaseOption),
	}, nil
}
