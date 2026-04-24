package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMSKClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_msk_cluster",
		CoreRFunc:           NewMSKCluster,
		ReferenceAttributes: []string{"aws_appautoscaling_target.resource_id"},
	}
}
func NewMSKCluster(d *engine.ResourceSpec) engine.CatalogItem {
	targets := []*aws.AppAutoscalingTarget{}
	for _, ref := range d.References("aws_appautoscaling_target.resource_id") {
		targets = append(targets, newAppAutoscalingTarget(ref, ref.UsageData))
	}

	var brokerEBSVolumeSize int64
	if d.Get("broker_node_group_info.0.ebs_volume_size").Exists() {
		// terraform-provider-aws v4
		brokerEBSVolumeSize = d.Get("broker_node_group_info.0.ebs_volume_size").Int()
	} else {
		// terraform-provider-aws v5
		brokerEBSVolumeSize = d.Get("broker_node_group_info.0.storage_info.0.ebs_storage_info.0.volume_size").Int()
	}

	r := &aws.MSKCluster{
		Address:                 d.Address,
		Region:                  d.Get("region").String(),
		BrokerNodes:             d.Get("number_of_broker_nodes").Int(),
		BrokerNodeInstanceType:  d.Get("broker_node_group_info.0.instance_type").String(),
		BrokerNodeEBSVolumeSize: brokerEBSVolumeSize,
		AppAutoscalingTarget:    targets,
	}
	return r
}
