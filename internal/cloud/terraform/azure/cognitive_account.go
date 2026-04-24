package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

func getCognitiveAccountRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_cognitive_account",
		CoreRFunc: newCognitiveAccount,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newCognitiveAccount(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	kind := d.Get("kind").String()

	if strings.EqualFold(kind, "speechservices") {
		return &azure.CognitiveAccountSpeech{
			Address: d.Address,
			Region:  region,
			Sku:     d.Get("sku_name").String(),
		}
	}

	if strings.EqualFold(kind, "luis") {
		return &azure.CognitiveAccountLUIS{
			Address: d.Address,
			Region:  region,
			Sku:     d.Get("sku_name").String(),
		}
	}

	if strings.EqualFold(kind, "textanalytics") {
		return &azure.CognitiveAccountLanguage{
			Address: d.Address,
			Region:  region,
			Sku:     d.Get("sku_name").String(),
		}
	}

	if strings.EqualFold(kind, "openai") {
		// OpenAI costs are counted as part of a Cognitive Deployment so
		// this resource is counted as free
		return engine.BlankCoreResource{
			Name: d.Address,
			Type: d.Type,
		}
	}

	logging.Logger.Warn().Msgf("Skipping resource %s. Kind %q is not supported", d.Address, kind)

	return nil
}
