package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2TrafficMirrorSessionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_traffic_mirror_session",
		CoreRFunc: NewEC2TrafficMirrorSession,
	}
}
func NewEC2TrafficMirrorSession(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EC2TrafficMirrorSession{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
