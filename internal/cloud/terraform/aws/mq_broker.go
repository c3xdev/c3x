package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMQBrokerRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_mq_broker",
		CoreRFunc: NewMQBroker,
	}
}
func NewMQBroker(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.MQBroker{
		Address:          d.Address,
		Region:           d.Get("region").String(),
		EngineType:       d.Get("engine_type").String(),
		HostInstanceType: d.Get("host_instance_type").String(),
		StorageType:      d.Get("storage_type").String(),
		DeploymentMode:   d.Get("deployment_mode").String(),
	}
	return r
}
