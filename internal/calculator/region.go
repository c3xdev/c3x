package calculator

import (
	"regexp"

	"github.com/c3xdev/c3x/internal/domain"
)

// Region-name shapes per provider, loose enough to accept any real
// region and strict enough to tell the three apart.
var providerRegionShape = map[string]*regexp.Regexp{
	"aws":   regexp.MustCompile(`^[a-z]{2}(-gov|-iso[a-z]*)?-[a-z]+-\d+$`), // eu-west-1
	"azure": regexp.MustCompile(`^[a-z]+[a-z0-9]*$`),                       // westeurope, eastus2
	"gcp":   regexp.MustCompile(`^[a-z]+-[a-z]+\d+$`),                      // europe-west4
}

// referenceRegions are the regions the catalog's filters are written
// against, used when neither the resource nor the configuration names a
// region for its provider.
var referenceRegions = map[string]string{
	"aws":   "us-east-1",
	"azure": "eastus",
	"gcp":   "us-central1",
}

// regionFor returns the region to price r in. The resource's own region
// wins. Otherwise the configured default is used only if it is a region
// of the resource's provider: a single --region (or the us-east-1
// default) must not send an Azure or GCP lookup to an AWS region name,
// which matched nothing and fell back with a misleading caveat.
func (e *Engine) regionFor(r domain.Resource, provider string) string {
	if r.Region != nil && *r.Region != "" {
		return *r.Region
	}
	shape, known := providerRegionShape[provider]
	if !known || (e.defaultRegion != "" && shape.MatchString(e.defaultRegion)) {
		return e.defaultRegion
	}
	return referenceRegions[provider]
}
