package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSecretManagerSecretVersionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_secret_manager_secret_version",
		CoreRFunc: newSecretManagerSecretVersion,
		ReferenceAttributes: []string{
			"secret",
		},
	}
}

func newSecretManagerSecretVersion(d *engine.ResourceSpec) engine.CatalogItem {
	replicasCount := int64(1)

	secretReferences := d.References("secret")
	if len(secretReferences) > 0 {
		replicasCount = secretManagerSecretReplicasCount(secretReferences[0])
	}

	return &google.SecretManagerSecretVersion{
		Address:              d.Address,
		Region:               d.Get("region").String(),
		ReplicationLocations: replicasCount,
	}
}
