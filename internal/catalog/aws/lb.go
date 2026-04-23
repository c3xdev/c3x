package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type LB struct {
	Address           string
	LoadBalancerType  string
	Region            string
	RuleEvaluations   *int64   `c3x_usage:"rule_evaluations"`
	NewConnections    *int64   `c3x_usage:"new_connections"`
	ActiveConnections *int64   `c3x_usage:"active_connections"`
	ProcessedBytesGB  *float64 `c3x_usage:"processed_bytes_gb"`
}

var LBUsageSchema = []*engine.ConsumptionField{
	{Key: "rule_evaluations", ValueType: engine.Int64, DefaultValue: 0},
	{Key: "new_connections", ValueType: engine.Int64, DefaultValue: 0},
	{Key: "active_connections", ValueType: engine.Int64, DefaultValue: 0},
	{Key: "processed_bytes_gb", ValueType: engine.Float64, DefaultValue: 0},
}

func (r *LB) CoreType() string {
	return "LB"
}

func (r *LB) UsageSchema() []*engine.ConsumptionField {
	return LBUsageSchema
}

func (r *LB) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *LB) BuildResource() *engine.Estimate {
	var maxLCU *decimal.Decimal

	var newConnectionsLCU *decimal.Decimal
	if r.NewConnections != nil {
		newConnections := decimal.NewFromInt(*r.NewConnections)
		newConnectionsLCU = decimalPtr(newConnections.Div(decimal.NewFromInt(100)))
		maxLCU = newConnectionsLCU
	}

	var activeConnectionsLCU *decimal.Decimal
	if r.ActiveConnections != nil {
		activeConnections := decimal.NewFromInt(*r.ActiveConnections)
		activeConnectionsLCU = decimalPtr(activeConnections.Div(decimal.NewFromInt(3000)))

		if maxLCU == nil {
			maxLCU = activeConnectionsLCU
		} else {
			maxLCU = decimalPtr(decimal.Max(*maxLCU, *activeConnectionsLCU))
		}
	}

	var processedBytesLCU *decimal.Decimal
	if r.ProcessedBytesGB != nil {
		processedBytes := decimal.NewFromFloat(*r.ProcessedBytesGB)
		processedBytesLCU = decimalPtr(processedBytes.Div(decimal.NewFromInt(1)))

		if maxLCU == nil {
			maxLCU = processedBytesLCU
		} else {
			maxLCU = decimalPtr(decimal.Max(*maxLCU, *processedBytesLCU))
		}
	}

	var costComponents []*engine.LineItem

	if strings.ToLower(r.LoadBalancerType) == "application" {
		var ruleEvaluationsLCU decimal.Decimal
		if r.RuleEvaluations != nil && maxLCU != nil {
			ruleEvaluations := decimal.NewFromInt(*r.RuleEvaluations)
			ruleEvaluationsLCU = ruleEvaluations.Div(decimal.NewFromInt(1000))

			if maxLCU == nil {
				maxLCU = &ruleEvaluationsLCU
			} else {
				maxLCU = decimalPtr(decimal.Max(*maxLCU, ruleEvaluationsLCU))
			}
		}

		costComponents = r.applicationLBCostComponents(maxLCU)
	} else {
		costComponents = r.networkLBCostComponents(maxLCU)
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *LB) applicationLBCostComponents(maxLCU *decimal.Decimal) []*engine.LineItem {
	productFamily := "Load Balancer-Application"

	return []*engine.LineItem{
		{
			Name:           "Application load balancer",
			Unit:           "hours",
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			UnitMultiplier: decimal.NewFromInt(1),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AWSELB"),
				ProductFamily: strPtr("Load Balancer-Application"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "locationType", Value: strPtr("AWS Region")},
					{Key: "usagetype", ValueRegex: regexPtr("^([A-Z]{3}\\d-|Global-|EU-)?LoadBalancerUsage$")},
				},
			},
		},
		r.capacityUnitsCostComponent(productFamily, maxLCU),
	}
}

func (r *LB) networkLBCostComponents(maxLCU *decimal.Decimal) []*engine.LineItem {
	productFamily := "Load Balancer-Network"

	return []*engine.LineItem{
		{
			Name:           "Network load balancer",
			Unit:           "hours",
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			UnitMultiplier: decimal.NewFromInt(1),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AWSELB"),
				ProductFamily: strPtr("Load Balancer-Network"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "locationType", Value: strPtr("AWS Region")},
					{Key: "usagetype", ValueRegex: strPtr("/LoadBalancerUsage/")},
				},
			},
		},
		r.capacityUnitsCostComponent(productFamily, maxLCU),
	}
}

func (r *LB) capacityUnitsCostComponent(productFamily string, maxLCU *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Load balancer capacity units",
		Unit:            "LCU",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: maxLCU,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSELB"),
			ProductFamily: strPtr(productFamily),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "locationType", Value: strPtr("AWS Region")},
				{Key: "usagetype", ValueRegex: strPtr("/^([A-Z]{3}\\d-|Global-|EU-)?LCUUsage/")},
			},
		},
		UsageBased: true,
	}
}
