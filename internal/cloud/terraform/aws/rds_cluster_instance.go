package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRDSClusterInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_rds_cluster_instance",
		CoreRFunc:           NewRDSClusterInstance,
		ReferenceAttributes: []string{"cluster_identifier"},
	}
}

func NewRDSClusterInstance(d *engine.ResourceSpec) engine.CatalogItem {
	piEnabled := d.Get("performance_insights_enabled").Bool()
	piLongTerm := piEnabled && d.Get("performance_insights_retention_period").Int() > 7

	ioOptimized := false
	clusterRefs := d.References("cluster_identifier")
	if len(clusterRefs) > 0 {
		ioOptimized = clusterRefs[0].Get("storage_type").String() == "aurora-iopt1"
	}

	r := &aws.RDSClusterInstance{
		Address:                              d.Address,
		Region:                               d.Get("region").String(),
		InstanceClass:                        d.Get("instance_class").String(),
		IOOptimized:                          ioOptimized,
		Engine:                               d.Get("engine").String(),
		Version:                              d.Get("engine_version").String(),
		PerformanceInsightsEnabled:           piEnabled,
		PerformanceInsightsLongTermRetention: piLongTerm,
	}
	return r
}
