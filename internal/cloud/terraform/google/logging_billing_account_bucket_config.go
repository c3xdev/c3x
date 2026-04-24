package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoggingBillingAccountBucketConfigRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_logging_billing_account_bucket_config",
		CoreRFunc: NewLoggingBillingAccountBucketConfig,
	}
}

func NewLoggingBillingAccountBucketConfig(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.Logging{
		Address: d.Address,
	}

	return r
}
