package recommend

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
)

var azureVMGenerationUpgrades = map[string]string{
	"Standard_D2_v3":  "Standard_D2_v5",
	"Standard_D4_v3":  "Standard_D4_v5",
	"Standard_D8_v3":  "Standard_D8_v5",
	"Standard_D16_v3": "Standard_D16_v5",
	"Standard_D2_v4":  "Standard_D2_v5",
	"Standard_D4_v4":  "Standard_D4_v5",
	"Standard_D8_v4":  "Standard_D8_v5",
	"Standard_D16_v4": "Standard_D16_v5",
	"Standard_E2_v3":  "Standard_E2_v5",
	"Standard_E4_v3":  "Standard_E4_v5",
	"Standard_E8_v3":  "Standard_E8_v5",
	"Standard_E2_v4":  "Standard_E2_v5",
	"Standard_E4_v4":  "Standard_E4_v5",
	"Standard_E8_v4":  "Standard_E8_v5",
	"Standard_F2_v2":  "Standard_F2s_v2",
}

func analyzeAzure(resource *engine.Estimate) []Recommendation {
	var recs []Recommendation

	switch resource.ResourceType {
	case "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine", "azurerm_virtual_machine":
		recs = append(recs, checkAzureVMGeneration(resource)...)
	case "azurerm_managed_disk":
		recs = append(recs, checkAzureDiskType(resource)...)
	}

	return recs
}

func checkAzureVMGeneration(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if !strings.Contains(cc.Name, "Instance usage") {
			continue
		}

		// Extract VM size from name like "Instance usage (pay as you go, Standard_D2_v3)"
		parts := strings.Split(cc.Name, ", ")
		if len(parts) < 2 {
			continue
		}
		vmSize := strings.TrimSuffix(parts[len(parts)-1], ")")

		if newSize, ok := azureVMGenerationUpgrades[vmSize]; ok {
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "instance-generation",
				Title:        fmt.Sprintf("Upgrade VM size (%s → %s)", vmSize, newSize),
				Description:  fmt.Sprintf("%s offers better price-performance than %s with newer hardware.", newSize, vmSize),
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

func checkAzureDiskType(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if strings.Contains(cc.Name, "Standard HDD") {
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "storage-type",
				Title:        "Consider upgrading from Standard HDD to Standard SSD",
				Description:  "Standard SSD offers better consistency and latency than Standard HDD with marginal cost increase.",
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}
