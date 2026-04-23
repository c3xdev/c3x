package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getConfigurationRecorderItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_config_configuration_recorder",
		CoreRFunc: NewConfigConfigurationRecorder,
	}
}
func NewConfigConfigurationRecorder(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.ConfigConfigurationRecorder{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
