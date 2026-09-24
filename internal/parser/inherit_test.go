package parser

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
)

func res(kind, name string, attrs map[string]any) domain.Resource {
	return domain.Resource{Ref: domain.Reference{Kind: kind, Name: name}, Attributes: attrs}
}

func attrOf(t *testing.T, rs []domain.Resource, name, attr string) any {
	t.Helper()
	for _, r := range rs {
		if r.Ref.Name == name {
			return r.Attributes[attr]
		}
	}
	t.Fatalf("resource %q not found", name)
	return nil
}

// TestInheritCopiesClusterStorageType is the case behind c3xdev/c3x#68:
// an Aurora instance's hourly rate depends on storage_type, which
// Terraform declares on the parent cluster.
func TestInheritCopiesClusterStorageType(t *testing.T) {
	t.Parallel()
	rs := []domain.Resource{
		res("aws_rds_cluster", "main", map[string]any{
			"cluster_identifier": "demo",
			"storage_type":       "aurora-iopt1",
		}),
		res("aws_rds_cluster_instance", "one", map[string]any{
			"cluster_identifier": "demo",
			"instance_class":     "db.r6g.large",
		}),
	}
	applyInheritance(rs)
	if got := attrOf(t, rs, "one", "storage_type"); got != "aurora-iopt1" {
		t.Errorf("instance storage_type = %v, want aurora-iopt1", got)
	}
}

// An explicit value on the child must win: inheritance fills gaps, it
// never overrides what the configuration actually says.
func TestInheritNeverOverwritesChildValue(t *testing.T) {
	t.Parallel()
	rs := []domain.Resource{
		res("aws_rds_cluster", "main", map[string]any{
			"cluster_identifier": "demo", "storage_type": "aurora-iopt1",
		}),
		res("aws_rds_cluster_instance", "one", map[string]any{
			"cluster_identifier": "demo", "storage_type": "",
		}),
	}
	applyInheritance(rs)
	if got := attrOf(t, rs, "one", "storage_type"); got != "" {
		t.Errorf("explicit empty storage_type was overwritten with %v", got)
	}
}

// Only the instance belonging to the matching cluster inherits.
func TestInheritMatchesTheRightParent(t *testing.T) {
	t.Parallel()
	rs := []domain.Resource{
		res("aws_rds_cluster", "opt", map[string]any{
			"cluster_identifier": "optimized", "storage_type": "aurora-iopt1",
		}),
		res("aws_rds_cluster", "std", map[string]any{
			"cluster_identifier": "standard",
		}),
		res("aws_rds_cluster_instance", "a", map[string]any{"cluster_identifier": "optimized"}),
		res("aws_rds_cluster_instance", "b", map[string]any{"cluster_identifier": "standard"}),
	}
	applyInheritance(rs)
	if got := attrOf(t, rs, "a", "storage_type"); got != "aurora-iopt1" {
		t.Errorf("instance a = %v, want aurora-iopt1", got)
	}
	if got := attrOf(t, rs, "b", "storage_type"); got != nil {
		t.Errorf("instance b inherited %v from the wrong cluster", got)
	}
}

// No parent, no join value, or an unresolved (non-string) reference must
// all be no-ops rather than errors. HCL input can produce any of them.
func TestInheritDegradesQuietly(t *testing.T) {
	t.Parallel()
	cases := map[string][]domain.Resource{
		"orphan instance": {
			res("aws_rds_cluster_instance", "one", map[string]any{"cluster_identifier": "demo"}),
		},
		"instance without a join value": {
			res("aws_rds_cluster", "main", map[string]any{
				"cluster_identifier": "demo", "storage_type": "aurora-iopt1",
			}),
			res("aws_rds_cluster_instance", "one", map[string]any{}),
		},
		"unresolved reference": {
			res("aws_rds_cluster", "main", map[string]any{
				"cluster_identifier": "demo", "storage_type": "aurora-iopt1",
			}),
			res("aws_rds_cluster_instance", "one", map[string]any{"cluster_identifier": nil}),
		},
		"parent has nothing to give": {
			res("aws_rds_cluster", "main", map[string]any{"cluster_identifier": "demo"}),
			res("aws_rds_cluster_instance", "one", map[string]any{"cluster_identifier": "demo"}),
		},
	}
	for name, rs := range cases {
		t.Run(name, func(t *testing.T) {
			applyInheritance(rs) // must not panic
			if got := attrOf(t, rs, "one", "storage_type"); got != nil {
				t.Errorf("unexpectedly inherited %v", got)
			}
		})
	}
}

func TestInheritHandlesEmptyAndSingleSets(t *testing.T) {
	t.Parallel()
	applyInheritance(nil)
	applyInheritance([]domain.Resource{})
	applyInheritance([]domain.Resource{res("aws_rds_cluster", "main", map[string]any{})})
}

// A db.serverless instance bills ACU-hours within the range declared on the
// cluster's serverlessv2_scaling_configuration block, so the block is
// copied across whole.
func TestInheritCopiesServerlessV2Scaling(t *testing.T) {
	t.Parallel()
	scaling := map[string]any{"min_capacity": 2.0, "max_capacity": 16.0}
	rs := []domain.Resource{
		res("aws_rds_cluster", "main", map[string]any{
			"cluster_identifier":                 "demo",
			"serverlessv2_scaling_configuration": scaling,
		}),
		res("aws_rds_cluster_instance", "one", map[string]any{
			"cluster_identifier": "demo",
			"instance_class":     "db.serverless",
		}),
	}
	applyInheritance(rs)
	got, ok := attrOf(t, rs, "one", "serverlessv2_scaling_configuration").(map[string]any)
	if !ok || got["min_capacity"] != 2.0 {
		t.Errorf("instance scaling = %v, want %v", got, scaling)
	}
}

// Cosmos DB children name their account by account_name, which matches the
// account's name attribute rather than an attribute of the same name.
func TestInheritCosmosJoinsOnAccountName(t *testing.T) {
	t.Parallel()
	geo := []any{
		map[string]any{"location": "eastus"},
		map[string]any{"location": "westus"},
	}
	rs := []domain.Resource{
		res("azurerm_cosmosdb_account", "a", map[string]any{
			"name": "acct-a", "geo_location": geo, "multiple_write_locations_enabled": true,
		}),
		res("azurerm_cosmosdb_account", "b", map[string]any{"name": "acct-b"}),
		res("azurerm_cosmosdb_sql_container", "one", map[string]any{"account_name": "acct-a"}),
		res("azurerm_cosmosdb_sql_container", "two", map[string]any{"account_name": "acct-b"}),
	}
	applyInheritance(rs)
	if got, _ := attrOf(t, rs, "one", "geo_location").([]any); len(got) != 2 {
		t.Errorf("container one geo_location = %v, want two regions", got)
	}
	if got := attrOf(t, rs, "one", "multiple_write_locations_enabled"); got != true {
		t.Errorf("container one multiple_write_locations_enabled = %v, want true", got)
	}
	if got := attrOf(t, rs, "two", "geo_location"); got != nil {
		t.Errorf("container two inherited %v from the wrong account", got)
	}
}

// The usual way to write the join is a reference the parser cannot
// evaluate (cluster_identifier = aws_rds_cluster.main.id). With exactly one
// candidate parent the child links to it; with several, nothing is guessed.
func TestInheritUnresolvedJoinUsesTheOnlyParent(t *testing.T) {
	t.Parallel()
	child := func() domain.Resource {
		r := res("aws_rds_cluster_instance", "one", map[string]any{"cluster_identifier": nil})
		r.Unresolved = []string{"cluster_identifier"}
		return r
	}

	single := []domain.Resource{
		res("aws_rds_cluster", "main", map[string]any{"storage_type": "aurora-iopt1"}),
		child(),
	}
	applyInheritance(single)
	if got := attrOf(t, single, "one", "storage_type"); got != "aurora-iopt1" {
		t.Errorf("single parent: storage_type = %v, want aurora-iopt1", got)
	}

	several := []domain.Resource{
		res("aws_rds_cluster", "a", map[string]any{"storage_type": "aurora-iopt1"}),
		res("aws_rds_cluster", "b", map[string]any{}),
		child(),
	}
	applyInheritance(several)
	if got := attrOf(t, several, "one", "storage_type"); got != nil {
		t.Errorf("several parents: guessed storage_type %v", got)
	}
}
