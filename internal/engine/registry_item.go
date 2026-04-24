package engine

// ReferenceIDFunc is used to let references be built using non-standard IDs (anything other than d.Get("id").string)
type ReferenceIDFunc func(d *ResourceSpec) []string

// CloudResourceIDFunc is used to calculate the cloud resource ids (AWS ARN, Google HREF, etc...) associated with the resource
type CloudResourceIDFunc func(d *ResourceSpec) []string

// RegionLookupFunc is used to look up the region of a resource, this is used to
// calculate the region of a resource if the region requires a lookup from
// reference attributes.
type RegionLookupFunc func(defaultRegion string, d *ResourceSpec) string

type CatalogEntry struct {
	Name                string
	Notes               []string
	RFunc               LegacyBuilder
	CoreRFunc           CatalogBuilder
	ReferenceAttributes []string
	CustomRefIDFunc     ReferenceIDFunc
	DefaultRefIDFunc    ReferenceIDFunc
	CloudResourceIDFunc CloudResourceIDFunc
	NoPrice             bool
	// GetRegion is used to look up the region of a resource if it has a region that
	// cannot be calculated from the default resource/provider data. If the GetRegion
	// is nil or the return result is empty the region will be calculated from the
	// default resource/provider data.
	GetRegion RegionLookupFunc
}
