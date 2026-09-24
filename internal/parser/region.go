package parser

import (
	"regexp"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
)

var (
	gcpRegion = regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]+$`)
	gcpZone   = regexp.MustCompile(`^([a-z]+-[a-z]+[0-9]+)-[a-z]$`)
)

// applyResourceRegions sets each resource's region from its own
// attributes where the resource declares one, which wins over the
// provider-level default the parsers assign. Azure has no provider-level
// region at all: every azurerm resource carries `location`, and before
// this every one of them was priced in the reference region, whatever
// its location said. GCP resources often name a zone rather than a
// region.
func applyResourceRegions(resources []domain.Resource) {
	for i := range resources {
		if region := regionFromAttributes(resources[i].Ref.Kind, resources[i].Attributes); region != "" {
			resources[i].Region = &region
		}
	}
}

// regionFromAttributes returns the pricing region a resource declares for
// itself, or "" when it declares none (or one that can't be a region,
// such as an unresolved value or a GCS multi-region like "US").
func regionFromAttributes(kind string, attrs map[string]any) string {
	str := func(key string) string {
		s, _ := attrs[key].(string)
		return strings.TrimSpace(s)
	}
	switch {
	case strings.HasPrefix(kind, "azurerm_"):
		// "West Europe" and "westeurope" are the same location; the price
		// data uses the second form.
		return strings.ToLower(strings.ReplaceAll(str("location"), " ", ""))
	case strings.HasPrefix(kind, "google_"):
		for _, key := range []string{"region", "zone", "location"} {
			v := strings.ToLower(str(key))
			if gcpRegion.MatchString(v) {
				return v
			}
			if m := gcpZone.FindStringSubmatch(v); m != nil {
				return m[1]
			}
		}
		return ""
	case strings.HasPrefix(kind, "aws_"):
		// AWS provider v6 lets a resource override the provider's region.
		return str("region")
	}
	return ""
}
