package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

// This resource is superseded by azurerm_data_factory_integration_runtime_azure_ssis
// in Terraform. Their instance types look the same, but the pricing page
// additionally mentions other operations for managed runtime.
func getDataFactoryIntegrationRuntimeManagedRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_data_factory_integration_runtime_managed",
		CoreRFunc: newDataFactoryIntegrationRuntimeManaged,
		ReferenceAttributes: []string{
			"data_factory_id",
			"data_factory_name",
			"resource_group_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			region := lookupRegion(d, []string{"resource_group_name", "data_factory_id", "data_factory_name"})

			dataFactoryIdRefs := d.References("data_factory_id")
			if region == "" && len(dataFactoryIdRefs) > 0 {
				region = lookupRegion(dataFactoryIdRefs[0], []string{"resource_group_name"})
			}

			// Old provider versions < 3 can reference data_factory_name
			dataFactoryNameRefs := d.References("data_factory_name")
			if region == "" && len(dataFactoryNameRefs) > 0 {
				region = lookupRegion(dataFactoryNameRefs[0], []string{"resource_group_name"})
			}
			return region
		},
	}
}

func newDataFactoryIntegrationRuntimeManaged(d *engine.ResourceSpec) engine.CatalogItem {
	licenseType := d.GetStringOrDefault("license_type", "LicenseIncluded")
	licenseIncluded := strings.EqualFold(licenseType, "LicenseIncluded")

	edition := d.GetStringOrDefault("edition", "Standard")
	enterprise := strings.EqualFold(edition, "Enterprise")

	nodes := d.GetInt64OrDefault("number_of_nodes", 1)

	nodeType := d.Get("node_size").String()
	instanceType := strings.ReplaceAll(nodeType, "Standard_", "")
	instanceType = strings.ReplaceAll(instanceType, "_", " ")

	r := &azure.DataFactoryIntegrationRuntimeManaged{
		Address:         d.Address,
		Region:          d.Region,
		Enterprise:      enterprise,
		LicenseIncluded: licenseIncluded,
		Instances:       nodes,
		InstanceType:    instanceType,
	}
	return r
}
