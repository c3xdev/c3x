package google

import (
	"github.com/c3xdev/c3x/internal/cloud/terraform/provider_schemas"
	"github.com/c3xdev/c3x/internal/engine"
)

var DefaultProviderRegion = "us-central1"

func GetDefaultRefIDFunc(d *engine.ResourceSpec) []string {

	defaultRefs := []string{d.Get("id").String()}

	if d.Get("self_link").Exists() {
		defaultRefs = append(defaultRefs, d.Get("self_link").String())
	}

	return defaultRefs
}

func DefaultCloudResourceIDFunc(d *engine.ResourceSpec) []string {
	return []string{}
}

func GetSpecialContext(d *engine.ResourceSpec) map[string]interface{} {
	return map[string]interface{}{}
}

func GetResourceRegion(d *engine.ResourceSpec) string {
	v := d.RawValues

	if v.Get("region").Exists() && v.Get("region").String() != "" {
		return v.Get("region").String()
	}

	return ""
}

func ParseTags(r *engine.ResourceSpec, externalTags, defaultLabels map[string]string) (map[string]string, []string) {

	_, supportsLabels := provider_schemas.GoogleLabelsSupport[r.Type]
	rLabels := r.Get("labels").Map()

	_, supportsUserLabels := provider_schemas.GoogleUserLabelsSupport[r.Type]
	rUserLabels := r.Get("user_labels").Map()

	_, supportsSettingsUserLabels := provider_schemas.GoogleSettingsUserLabelsSupport[r.Type]
	rSettingsUserLabels := r.Get("settings.0.user_labels").Map()

	missingForLabels := engine.ExtractMissingVarsCausingMissingAttributeKeys(r, "labels")
	missingForUserLabels := engine.ExtractMissingVarsCausingMissingAttributeKeys(r, "user_labels")
	missingForSettingsUserLabels := engine.ExtractMissingVarsCausingMissingAttributeKeys(r, "settings.0.user_labels")
	missing := append(append(missingForLabels, missingForUserLabels...), missingForSettingsUserLabels...)

	if !supportsLabels && len(rLabels) == 0 &&
		!supportsUserLabels && len(rUserLabels) == 0 &&
		!supportsSettingsUserLabels && len(rSettingsUserLabels) == 0 {
		return nil, missing
	}

	tags := make(map[string]string)

	for k, v := range defaultLabels {
		tags[k] = v
	}
	for k, v := range rLabels {
		tags[k] = v.String()
	}
	for k, v := range rUserLabels {
		tags[k] = v.String()
	}
	for k, v := range rSettingsUserLabels {
		tags[k] = v.String()
	}
	for k, v := range externalTags {
		tags[k] = v
	}

	return tags, missing
}
