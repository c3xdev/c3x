package parser

import "github.com/c3xdev/c3x/internal/domain"

// Some costs are decided by an attribute that Terraform declares on a
// different resource than the one billed for it. Aurora is the case that
// prompted this: the hourly rate of an aws_rds_cluster_instance depends on
// storage_type, which is set on the parent aws_rds_cluster.
//
// The attribute is copied onto the child here, at parse time, rather than
// read across resources at pricing time. That placement is deliberate. The
// catalog is fetched remote-first by every client, including releases whose
// expression engine is older than whatever the catalog uses, so an
// expression that reaches across resources would fail outright on them. A
// client-side enrichment degrades instead of breaking: an older client
// simply does not copy the attribute and prices the resource the way it
// always did.
//
// Inheritance never overwrites. A value the resource declares itself always
// wins, so this cannot mask an explicit configuration.
type inheritRule struct {
	// Child is the resource kind that gains the attributes.
	Child string
	// Parent is the kind they come from.
	Parent string
	// Join is the child attribute that links it to its parent (for
	// Aurora, cluster_identifier).
	Join string
	// ParentJoin is the parent attribute Join must equal. Empty means the
	// parent carries it under the same name as the child.
	ParentJoin string
	// Attrs are the attribute names copied from parent to child.
	Attrs []string
}

// cosmosThroughputKinds are the Cosmos DB resources that can carry their
// own provisioned throughput. Throughput bills once per account region, at
// a dearer rate when the account accepts writes in several regions, and
// both of those are declared on the account.
var cosmosThroughputKinds = []string{
	"azurerm_cosmosdb_sql_database",
	"azurerm_cosmosdb_sql_container",
	"azurerm_cosmosdb_mongo_database",
	"azurerm_cosmosdb_mongo_collection",
	"azurerm_cosmosdb_cassandra_keyspace",
	"azurerm_cosmosdb_gremlin_database",
	"azurerm_cosmosdb_gremlin_graph",
	"azurerm_cosmosdb_table",
}

var inheritRules = func() []inheritRule {
	rules := []inheritRule{
		{
			Child:  "aws_rds_cluster_instance",
			Parent: "aws_rds_cluster",
			Join:   "cluster_identifier",
			// serverlessv2_scaling_configuration carries the ACU range a
			// db.serverless instance bills against.
			Attrs: []string{"storage_type", "serverlessv2_scaling_configuration"},
		},
	}
	for _, kind := range cosmosThroughputKinds {
		rules = append(rules, inheritRule{
			Child:      kind,
			Parent:     "azurerm_cosmosdb_account",
			Join:       "account_name",
			ParentJoin: "name",
			// multiple_write_locations_enabled is the azurerm 4.x name,
			// enable_multiple_write_locations the 3.x one.
			Attrs: []string{"geo_location", "multiple_write_locations_enabled", "enable_multiple_write_locations"},
		})
	}
	return rules
}()

// applyInheritance copies parent attributes onto children in place, per
// [inheritRules]. Resources with no rule, no join value, or no matching
// parent are left exactly as they were.
func applyInheritance(resources []domain.Resource) {
	if len(resources) < 2 {
		return
	}
	for _, rule := range inheritRules {
		parentJoin := rule.ParentJoin
		if parentJoin == "" {
			parentJoin = rule.Join
		}
		// Index the parents of this rule by their join value. Built per
		// rule rather than once, because the join attribute differs
		// between rules and the sets are small.
		parents := make(map[string]map[string]any)
		var all []map[string]any
		for _, r := range resources {
			if r.Ref.Kind != rule.Parent {
				continue
			}
			all = append(all, r.Attributes)
			if key, ok := attrString(r.Attributes, parentJoin); ok {
				parents[key] = r.Attributes
			}
		}
		if len(all) == 0 {
			continue
		}
		for i := range resources {
			child := &resources[i]
			if child.Ref.Kind != rule.Child {
				continue
			}
			parent := parentOf(child, rule.Join, parents, all)
			if parent == nil {
				continue
			}
			for _, name := range rule.Attrs {
				if _, present := child.Attributes[name]; present {
					continue // the child said it itself; leave it alone
				}
				if v, ok := parent[name]; ok && v != nil {
					if child.Attributes == nil {
						child.Attributes = make(map[string]any, len(rule.Attrs))
					}
					child.Attributes[name] = v
				}
			}
		}
	}
}

// parentOf finds the parent a child links to. A literal join value must
// match a parent's exactly. When the join is a reference the parser could
// not evaluate (cluster_identifier = aws_rds_cluster.main.id, the usual way
// to write it), the child still links to the parent if there is exactly one
// candidate: with a single cluster or account in the set there is nothing
// else the reference can point at. With several, nothing is guessed.
func parentOf(child *domain.Resource, join string, parents map[string]map[string]any, all []map[string]any) map[string]any {
	if key, ok := attrString(child.Attributes, join); ok {
		return parents[key]
	}
	if len(all) == 1 && isUnresolved(child, join) {
		return all[0]
	}
	return nil
}

// isUnresolved reports whether the configuration sets name on r but the
// parser could not evaluate it.
func isUnresolved(r *domain.Resource, name string) bool {
	for _, p := range r.Unresolved {
		if p == name {
			return true
		}
	}
	return false
}

// attrString reads an attribute as a non-empty string. Join values that are
// absent, nil, empty, or not a string (an unresolved HCL reference, say)
// cannot link anything, so they report false.
func attrString(attrs map[string]any, name string) (string, bool) {
	v, ok := attrs[name]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}
