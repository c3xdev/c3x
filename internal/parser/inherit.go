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
	// Join is the attribute both carry with the same value, which is what
	// links a child to its parent (for Aurora, cluster_identifier).
	Join string
	// Attrs are the attribute names copied from parent to child.
	Attrs []string
}

var inheritRules = []inheritRule{
	{
		Child:  "aws_rds_cluster_instance",
		Parent: "aws_rds_cluster",
		Join:   "cluster_identifier",
		Attrs:  []string{"storage_type"},
	},
}

// applyInheritance copies parent attributes onto children in place, per
// [inheritRules]. Resources with no rule, no join value, or no matching
// parent are left exactly as they were.
func applyInheritance(resources []domain.Resource) {
	if len(resources) < 2 {
		return
	}
	for _, rule := range inheritRules {
		// Index the parents of this rule by their join value. Built per
		// rule rather than once, because the join attribute differs
		// between rules and the sets are small.
		parents := make(map[string]map[string]any)
		for _, r := range resources {
			if r.Ref.Kind != rule.Parent {
				continue
			}
			if key, ok := attrString(r.Attributes, rule.Join); ok {
				parents[key] = r.Attributes
			}
		}
		if len(parents) == 0 {
			continue
		}
		for i := range resources {
			child := &resources[i]
			if child.Ref.Kind != rule.Child {
				continue
			}
			key, ok := attrString(child.Attributes, rule.Join)
			if !ok {
				continue
			}
			parent, ok := parents[key]
			if !ok {
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
