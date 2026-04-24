package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	eng "github.com/c3xdev/c3x/internal/engine"
)

type ElastiCacheReplicationGroup struct {
	Address                       string
	Region                        string
	NodeType                      string
	Engine                        string
	CacheClusters                 int64
	ClusterNodeGroups             int64
	ClusterReplicasPerNodeGroup   int64
	SnapshotRetentionLimit        int64
	SnapshotStorageSizeGB         *float64 `c3x_usage:"snapshot_storage_size_gb"`
	ReservedInstanceTerm          *string  `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption *string  `c3x_usage:"reserved_instance_payment_option"`

	AppAutoscalingTarget []*AppAutoscalingTarget
}

func (r *ElastiCacheReplicationGroup) CoreType() string {
	return "ElastiCacheReplicationGroup"
}

func (r *ElastiCacheReplicationGroup) UsageSchema() []*eng.ConsumptionField {
	return []*eng.ConsumptionField{
		{Key: "snapshot_storage_size_gb", ValueType: eng.Float64, DefaultValue: 0},
		{Key: "reserved_instance_term", DefaultValue: "", ValueType: eng.String},
		{Key: "reserved_instance_payment_option", DefaultValue: "", ValueType: eng.String},
	}
}

func (r *ElastiCacheReplicationGroup) PopulateUsage(u *eng.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ElastiCacheReplicationGroup) BuildResource() *eng.Estimate {
	engine := r.Engine
	if engine == "" {
		engine = "redis"
	}

	var autoscaling bool
	nodeGroups := r.ClusterNodeGroups
	replicasPerNodeGroup := r.ClusterReplicasPerNodeGroup
	for _, target := range r.AppAutoscalingTarget {
		switch target.ScalableDimension {
		case "elasticache:replication-group:NodeGroups":
			autoscaling = true
			if target.Capacity != nil {
				nodeGroups = *target.Capacity
			} else {
				nodeGroups = target.MinCapacity
			}
		case "elasticache:replication-group:Replicas":
			autoscaling = true
			if target.Capacity != nil {
				replicasPerNodeGroup = *target.Capacity
			} else {
				replicasPerNodeGroup = target.MinCapacity
			}
		}
	}

	cacheNodes := r.CacheClusters
	if nodeGroups > 0 {
		// CacheClusters is mutually exclusive with ClusterNodeGroups/ClusterReplicasPerNodeGroup
		cacheNodes = (nodeGroups * replicasPerNodeGroup) + nodeGroups
	}

	cluster := &ElastiCacheCluster{
		Region:                        r.Region,
		NodeType:                      r.NodeType,
		Engine:                        engine,
		CacheNodes:                    cacheNodes,
		SnapshotRetentionLimit:        r.SnapshotRetentionLimit,
		SnapshotStorageSizeGB:         r.SnapshotStorageSizeGB,
		ReservedInstanceTerm:          r.ReservedInstanceTerm,
		ReservedInstancePaymentOption: r.ReservedInstancePaymentOption,
	}

	costComponents := []*eng.LineItem{
		cluster.elastiCacheCostComponent(autoscaling),
	}

	if strings.ToLower(engine) == "redis" && r.SnapshotRetentionLimit > 1 {
		costComponents = append(costComponents, cluster.backupStorageCostComponent())
	}

	return &eng.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
