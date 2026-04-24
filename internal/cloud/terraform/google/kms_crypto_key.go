package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getKMSCryptoKeyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_kms_crypto_key",
		CoreRFunc: NewKMSCryptoKey,
	}
}
func NewKMSCryptoKey(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.KMSCryptoKey{Address: d.Address, Region: d.Get("region").String(), Algorithm: d.Get("version_template.0.algorithm").String(), ProtectionLevel: d.Get("version_template.0.protection_level").String(), RotationPeriod: d.Get("rotation_period").String(), VersionTemplate: d.Get("version_template").String()}
	return r
}
