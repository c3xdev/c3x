package azure

import (
	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

func GetAzureRMKeyVaultCertificateRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_key_vault_certificate",
		RFunc: NewAzureRMKeyVaultCertificate,
		ReferenceAttributes: []string{
			"key_vault_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"key_vault_id"})
		},
	}
}

func NewAzureRMKeyVaultCertificate(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	var costComponents []*engine.LineItem

	var skuName string
	keyVault := d.References("key_vault_id")
	if len(keyVault) > 0 {
		skuName = cases.Title(language.English).String(keyVault[0].Get("sku_name").String())
	} else {
		logging.Logger.Warn().Msgf("Skipping resource %s. Could not find its 'key_vault_id.sku_name' property.", d.Address)
		return nil
	}

	var certificateRenewals, certificateOperations *decimal.Decimal
	if u != nil && u.Get("monthly_certificate_renewal_requests").Exists() {
		certificateRenewals = decimalPtr(decimal.NewFromInt(u.Get("monthly_certificate_renewal_requests").Int()))
	}
	costComponents = append(costComponents, vaultKeysCostComponent(
		"Certificate renewals",
		region,
		"requests",
		skuName,
		"Certificate Renewal Request",
		"0",
		certificateRenewals,
		1))

	if u != nil && u.Get("monthly_certificate_other_operations").Exists() {
		certificateOperations = decimalPtr(decimal.NewFromInt(u.Get("monthly_certificate_other_operations").Int()))
	}
	costComponents = append(costComponents, vaultKeysCostComponent(
		"Certificate operations",
		region,
		"10K transactions",
		skuName,
		"Operations",
		"0",
		certificateOperations,
		10000))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
