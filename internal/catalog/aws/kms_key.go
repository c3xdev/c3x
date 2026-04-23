package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type KMSKey struct {
	Address               string
	Region                string
	CustomerMasterKeySpec string
}

func (r *KMSKey) CoreType() string {
	return "KMSKey"
}

func (r *KMSKey) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *KMSKey) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KMSKey) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.customerMasterKeyCostComponent(),
	}

	costComponents = append(costComponents, r.requestsCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *KMSKey) customerMasterKeyCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Customer master key",
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("awskms"),
			ProductFamily: strPtr("Encryption Key"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/KMS-Keys/")},
			},
		},
	}
}

func (r *KMSKey) requestsCostComponents() []*engine.LineItem {
	switch r.CustomerMasterKeySpec {
	case "RSA_2048":
		return []*engine.LineItem{
			r.requestsCostComponent("Requests (RSA 2048)", "/KMS-Requests-Asymmetric-RSA_2048/"),
		}
	case
		"RSA_3072",
		"RSA_4096",
		"ECC_NIST_P256",
		"ECC_NIST_P384",
		"ECC_NIST_P521",
		"ECC_SECG_P256K1":
		return []*engine.LineItem{
			r.requestsCostComponent("Requests (asymmetric)", "/KMS-Requests-Asymmetric$/"),
		}
	}

	return []*engine.LineItem{
		r.requestsCostComponent("Requests", "/KMS-Requests$/"),
		r.requestsCostComponent("ECC GenerateDataKeyPair requests", "/KMS-Requests-GenerateDatakeyPair-ECC/"),
		r.requestsCostComponent("RSA GenerateDataKeyPair requests", "/KMS-Requests-GenerateDatakeyPair-ECC/"),
	}
}

func (r *KMSKey) requestsCostComponent(name string, usagetype string) *engine.LineItem {
	return &engine.LineItem{
		Name:           name,
		Unit:           "10k requests",
		UnitMultiplier: decimal.NewFromInt(10000),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("awskms"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(usagetype)},
			},
		},
	}
}
