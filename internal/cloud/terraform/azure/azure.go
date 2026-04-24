package azure

import (
	"github.com/c3xdev/c3x/internal/cloud/terraform/provider_schemas"
	"github.com/c3xdev/c3x/internal/engine"
)

var DefaultProviderRegion = "eastus"

func GetDefaultRefIDFunc(d *engine.ResourceSpec) []string {
	return []string{d.Get("id").String()}
}

func DefaultCloudResourceIDFunc(d *engine.ResourceSpec) []string {
	return []string{}
}

func GetSpecialContext(d *engine.ResourceSpec) map[string]interface{} {
	return map[string]interface{}{}
}

func ParseTags(externalTags map[string]string, r *engine.ResourceSpec) (map[string]string, []string) {
	_, supportsTags := provider_schemas.AzureTagsSupport[r.Type]
	rTags := r.Get("tags").Map()
	missing := engine.ExtractMissingVarsCausingMissingAttributeKeys(r, "tags")
	if !supportsTags && len(rTags) == 0 {
		return nil, missing
	}
	tags := make(map[string]string)
	for k, v := range rTags {
		tags[k] = v.String()
	}
	for k, v := range externalTags {
		tags[k] = v
	}
	return tags, missing
}
