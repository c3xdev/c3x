package domain

import (
	"encoding/json"
	"fmt"
)

// Reference uniquely identifies a resource within an Estimate.
//
// Kind is the IaC type (e.g. "aws_instance", "azurerm_virtual_machine").
// Name is the address-suffix the IaC layer assigned — for Terraform that
// includes any module prefix and count/for_each suffix, e.g.
// `module.frontend.aws_instance.web[0]`.
type Reference struct {
	Kind string
	Name string
}

// Label renders the reference for display: "kind.name".
func (r Reference) Label() string { return r.Kind + "." + r.Name }

func (r Reference) String() string { return r.Label() }

// Resource is a parsed IaC resource with its attributes already resolved
// to literal values. The calculator turns Resources into Costs.
//
// Attributes are JSON because that's the lowest-common-denominator shape
// every parser (Terraform HCL, Terraform plan JSON, CloudFormation YAML)
// can produce. Catalog expressions then read attributes by name.
//
// Region is the cloud region this resource will deploy to, when
// determinable from the IaC source. A nil value means the calculator
// should fall back to the configured default region.
//
// Action is populated only when parsing plan JSON; it carries the
// Terraform-determined action for this resource (create, update, no-op).
// Consumers like the delta renderer can use it to show per-resource
// deltas without requiring a separate baseline file.
type Resource struct {
	Ref        Reference
	Attributes map[string]any
	Region     *string
	Action     PlanAction
}

// PlanAction describes what Terraform intends to do with a resource.
// Empty when the resource was parsed from HCL (no plan context).
type PlanAction string

const (
	PlanActionNone   PlanAction = ""       // HCL-parsed or unknown
	PlanActionNoOp   PlanAction = "no-op"  // unchanged
	PlanActionCreate PlanAction = "create" // new resource
	PlanActionUpdate PlanAction = "update" // in-place or with replacement
	PlanActionDelete PlanAction = "delete" // scheduled for removal
)

// AttrString reads an attribute as a string, returning ok=false if it's
// missing or not a string. Centralizing the type-assertions here keeps
// the calculator and recommender from re-implementing them.
func (r Resource) AttrString(key string) (string, bool) {
	v, ok := r.Attributes[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// AttrInt reads an attribute that may have arrived as int, int64, or
// float64 (JSON only ever decodes numbers as float64 unless told
// otherwise). Returns ok=false on missing or non-numeric values.
func (r Resource) AttrInt(key string) (int64, bool) {
	v, ok := r.Attributes[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

// AttrBool reads an attribute as a bool. Strings "true"/"false" are not
// accepted — Resource is post-parse, attributes should already be typed.
func (r Resource) AttrBool(key string) (bool, bool) {
	v, ok := r.Attributes[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// ResolveRegion returns the resource-specific region if set, otherwise
// `fallback`. Used by the calculator when building a PriceQuery so each
// callsite doesn't repeat the nil-check.
func (r Resource) ResolveRegion(fallback string) string {
	if r.Region != nil && *r.Region != "" {
		return *r.Region
	}
	return fallback
}

// String is intentionally non-trivial so debug logging shows something
// useful without the caller having to format manually.
func (r Resource) String() string {
	return fmt.Sprintf("%s(region=%v, attrs=%d)", r.Ref.Label(), r.Region, len(r.Attributes))
}
