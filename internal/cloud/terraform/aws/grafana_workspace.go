package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getGrafanaWorkspaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_grafana_workspace",
		CoreRFunc:           NewGrafanaWorkspace,
		ReferenceAttributes: []string{"aws_grafana_license_association.workspace_id"},
	}
}

func NewGrafanaWorkspace(d *engine.ResourceSpec) engine.CatalogItem {
	licenseType := "STANDARD"
	licenseAssoc := d.References("aws_grafana_license_association.workspace_id")
	if len(licenseAssoc) > 0 {
		licenseType = licenseAssoc[0].Get("license_type").String()
	}

	r := &aws.GrafanaWorkspace{
		Address: d.Address,
		Region:  d.Get("region").String(),
		License: licenseType,
	}

	return r
}
