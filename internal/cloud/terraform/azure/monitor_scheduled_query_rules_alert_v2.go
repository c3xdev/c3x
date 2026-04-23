package azure

import (
	duration "github.com/channelmeter/iso8601duration"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMonitorScheduledQueryRulesAlertV2RegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_monitor_scheduled_query_rules_alert_v2",
		CoreRFunc: newMonitorScheduledQueryRulesAlertV2,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMonitorScheduledQueryRulesAlertV2(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	freq := int64(1)
	ef, err := duration.FromString(d.Get("evaluation_frequency").String())
	if err != nil {
		logging.Logger.Warn().Str(
			"resource", d.Address,
		).Msgf("failed to parse ISO8061 duration string '%s' using 1 minute frequency", d.Get("evaluation_frequency").String())
	} else {
		freq = int64(ef.ToDuration().Minutes())
	}

	scopeCount := 1 // default scope is the azure subscription, so count == 1
	if !d.IsEmpty("scopes") {
		scopeCount = len(d.Get("scopes").Array())
	}

	criteriaDimensionsCount := 0
	for _, c := range d.Get("criteria").Array() {
		criteriaDimensionsCount += len(c.Get("dimension").Array())
	}

	return &azure.MonitorScheduledQueryRulesAlert{
		Address:          d.Address,
		Region:           region,
		Enabled:          d.GetBoolOrDefault("enabled", true),
		TimeSeriesCount:  int64(scopeCount * criteriaDimensionsCount),
		FrequencyMinutes: freq,
	}
}
