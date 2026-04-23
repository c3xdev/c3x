package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getFunctionAppRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "azurerm_function_app",
		ReferenceAttributes: []string{
			"app_service_plan_id",
		},
		CoreRFunc: func(d *engine.ResourceSpec) engine.CatalogItem {
			return newFunctionApp(d)
		},
	}
}

func newFunctionApp(d *engine.ResourceSpec) *azure.FunctionApp {
	appServicePlan := d.References("app_service_plan_id")
	servicePlan := d.References("service_plan_id")
	region := d.Region

	if len(appServicePlan) == 0 && len(servicePlan) == 0 {
		return &azure.FunctionApp{
			Address: d.Address,
			Region:  region,
			Tier:    "standard",
		}
	}

	if len(appServicePlan) > 0 {
		data := appServicePlan[0]
		return newFunctionAppFromAppServicePlanRef(d, data)
	}

	data := servicePlan[0]
	return newFunctionAppFromAppServicePlanRef(d, data)
}

func newFunctionAppFromAppServicePlanRef(d *engine.ResourceSpec, data *engine.ResourceSpec) *azure.FunctionApp {
	region := d.Region

	tier := "standard"
	// support for the legacy azurerm_app_service_plan resource. This is only applicable for the legacy azurerm_function_app resource.
	if data.Get("sku").Exists() {
		skuTier := strings.ToLower(data.Get("sku.0.tier").String())
		skuSize := strings.ToLower(data.Get("sku.0.size").String())
		kind := strings.ToLower(data.Get("kind").String())

		if strings.ToLower(skuSize) != "y1" && (strings.ToLower(kind) == "elastic" || strings.ToLower(skuTier) == "elasticpremium") {
			tier = "premium"
		}

		return &azure.FunctionApp{
			Address: d.Address,
			Region:  region,
			SKUName: skuSize,
			Tier:    tier,
			OSType:  kind,
		}
	}

	skuName := data.Get("sku_name").String()
	if strings.HasPrefix(strings.ToLower(skuName), "ep") {
		tier = "premium"
	}

	return &azure.FunctionApp{
		Address: d.Address,
		Region:  region,
		SKUName: strings.ToLower(skuName),
		Tier:    tier,
		OSType:  strings.ToLower(data.Get("os_type").String()),
	}
}
