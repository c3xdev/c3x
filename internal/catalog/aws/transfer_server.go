package aws

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// TransferServer defines a AWS Transfer Server resource from Transfer Family
// service. It supports multiple transfer protocols like FTP/FTPS/SFTP and
// each is billed hourly when enabled. It also bills the amount of data
// being downloaded/uploaded over those protocols.
//
// See more resource information here: https://aws.amazon.com/aws-transfer-family/.
//
// See the pricing information here: https://aws.amazon.com/aws-transfer-family/pricing/.
type TransferServer struct {
	Address   string
	Region    string
	Protocols []string

	// "usage" args
	MonthlyDataDownloadedGB *float64 `c3x_usage:"monthly_data_downloaded_gb"`
	MonthlyDataUploadedGB   *float64 `c3x_usage:"monthly_data_uploaded_gb"`
}

// TransferServerUsageSchema defines a list of usage items for TransferServer.
func (r *TransferServer) CoreType() string {
	return "TransferServer"
}

func (r *TransferServer) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_downloaded_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_data_uploaded_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the TransferServer.
// It uses the `c3x_usage` struct tags to populate data into the TransferServer.
func (r *TransferServer) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid TransferServer struct.
// This method is called after the resource is initialised by an IaC provider.
func (r *TransferServer) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	for _, protocol := range r.Protocols {
		costComponents = append(costComponents, r.protocolEnabledCostComponent(protocol))
	}

	costComponents = append(costComponents, r.dataDownloadedCostComponent())
	costComponents = append(costComponents, r.dataUploadedCostComponent())

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *TransferServer) protocolEnabledCostComponent(protocol string) *engine.LineItem {
	return &engine.LineItem{
		Name:           fmt.Sprintf("%s protocol enabled", protocol),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter:  r.buildProductFilter(protocol, "^[A-Z0-9]*-ProtocolHours$"),
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *TransferServer) dataDownloadedCostComponent() *engine.LineItem {
	// The pricing is identical for all protocols and the traffic is combined
	transferProtocol := "FTP"

	return &engine.LineItem{
		Name:            "Data downloaded",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataDownloadedGB),
		ProductFilter:   r.buildProductFilter(transferProtocol, "^[A-Z0-9]*-DownloadBytes$"),
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *TransferServer) dataUploadedCostComponent() *engine.LineItem {
	// The pricing is identical for all protocols and the traffic is combined
	transferProtocol := "FTP"

	return &engine.LineItem{
		Name:            "Data uploaded",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataUploadedGB),
		ProductFilter:   r.buildProductFilter(transferProtocol, "^[A-Z0-9]*-UploadBytes$"),
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *TransferServer) buildProductFilter(protocol, usageType string) *engine.ProductSelector {
	// The pricing for all storage types is identical, but for some protocols
	// EFS prices are missing in the pricing API.
	storageType := "S3"

	return &engine.ProductSelector{
		VendorName:    strPtr("aws"),
		Region:        strPtr(r.Region),
		Service:       strPtr("AWSTransfer"),
		ProductFamily: strPtr("AWS Transfer Family"),
		AttributeFilters: []*engine.AttributeMatch{
			{Key: "usagetype", ValueRegex: regexPtr(usageType)},
			{Key: "operation", ValueRegex: regexPtr(fmt.Sprintf("^%s:%s$", protocol, storageType))},
		},
	}
}
