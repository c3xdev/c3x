package aws

import (
	"regexp"
	"strings"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

var adReg = regexp.MustCompile(`(AD)`)

func getDirectoryServiceDirectory() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_directory_service_directory",
		CoreRFunc: newDirectoryServiceDirectory,
	}
}

func newDirectoryServiceDirectory(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	regionName, ok := aws.RegionMapping[region]
	if !ok {
		logging.Logger.Warn().Msgf("Could not find mapping for resource %s region %s", d.Address, region)
	}

	a := &aws.DirectoryServiceDirectory{
		Address:    d.Address,
		Region:     region,
		RegionName: regionName,
		Type:       getType(d.Get("type").String()),
		Edition:    d.Get("edition").String(),
		Size:       d.Get("size").String(),
	}

	return a
}

// getType returns the terraform directory type with AD spaced, e.g:
// MicrosoftAD => Microsoft AD
func getType(t string) string {
	return strings.TrimSpace(adReg.ReplaceAllString(t, " AD "))
}
