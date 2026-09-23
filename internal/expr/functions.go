// Package expr wraps github.com/expr-lang/expr with the c3x stdlib so
// catalog TOMLs can express quantity, rate, and when predicates in a
// small expression language.
//
// The wrapper exists so the rest of the engine never imports expr-lang
// directly. If we ever need to swap the evaluator (custom DSL, CEL, a
// fork), only this package changes.
package expr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// PriceLookup is the contract the calculator supplies so `price("name")`
// expressions can resolve mapping names to per-unit rates without
// pulling the pricing module into the expression layer.
type PriceLookup func(mappingName string) (decimal.Decimal, string, error)

// MonthlyHours is the conventional hours-per-month constant used by
// every catalog with hourly-billed resources. Matches the industry
// convention of treating a month as 730 hours (≈ 730.484 actual).
const MonthlyHours = 730

// stdFunctions assembles the function map injected into the expression
// environment. `lookup` is the per-evaluation price resolver supplied by
// the calculator; everything else is stateless.
//
// Functions return Go primitives (float64, bool, any) — expr-lang
// handles the conversion to/from typed values. Anywhere we'd want
// Decimal precision, we round-trip via shopspring/decimal at the caller.
func stdFunctions(lookup PriceLookup) map[string]any {
	return map[string]any{
		// price("mapping") → per-unit rate for that mapping. Returns 0
		// if the lookup yields no priced products (e.g. ACM public
		// certificates are free). The error path is reserved for genuine
		// failures (network, malformed mapping) which the calculator
		// surfaces as a degraded line item.
		"price": func(name string) (float64, error) {
			if lookup == nil {
				return 0, errors.New("price() called without a PriceLookup wired (calculator bug)")
			}
			d, _, err := lookup(name)
			if err != nil {
				return 0, fmt.Errorf("price(%q): %w", name, err)
			}
			f, _ := d.Float64()
			return f, nil
		},

		// default(value, fallback) → returns value if it's non-nil and
		// non-empty-string, otherwise fallback. Centralizes the "use the
		// resource's attribute or fall back to a sensible default" idiom.
		"default": func(v, fallback any) any {
			switch t := v.(type) {
			case nil:
				return fallback
			case string:
				if t == "" {
					return fallback
				}
			}
			return v
		},

		// monthly_hours() → the hours-per-month constant (730). Catalog
		// authors use this for any hourly-billed quantity so the constant
		// is named, not magic.
		"monthly_hours": func() int {
			return MonthlyHours
		},

		// sum_field(blocks, "size") → sums a numeric field across a
		// repeated Terraform block (`ebs_block_device` appears once
		// per device, parsed as []any of maps). nil / non-list /
		// missing fields contribute 0, so catalog expressions stay
		// total even when the block is absent.
		"sum_field": func(v any, field string) float64 {
			items, ok := v.([]any)
			if !ok {
				if m, isMap := v.(map[string]any); isMap {
					items = []any{m} // single block parsed unwrapped
				} else {
					return 0
				}
			}
			total := 0.0
			for _, it := range items {
				m, isMap := it.(map[string]any)
				if !isMap {
					continue
				}
				switch n := m[field].(type) {
				case float64:
					total += n
				case int:
					total += float64(n)
				case int64:
					total += float64(n)
				}
			}
			return total
		},

		// count_blocks(blocks) → number of occurrences of a repeated
		// block; 0 for nil/absent, 1 for a single unwrapped map.
		"count_blocks": func(v any) int {
			switch t := v.(type) {
			case []any:
				return len(t)
			case map[string]any:
				return 1
			default:
				return 0
			}
		},

		// pick(cond, a, b) → ternary as a function. Named `pick` because
		// `if` is a reserved keyword in expr-lang. Catalog authors who
		// prefer the native ternary `cond ? a : b` may use that instead.
		"pick": func(cond bool, a, b any) any {
			if cond {
				return a
			}
			return b
		},

		// replace(s, old, new) → string replace-all. Used by catalog
		// authors to normalise Terraform attribute values into the
		// shape the upstream pricing catalog stores. Example: an
		// Amazon MQ broker's `host_instance_type` arrives as
		// "mq.t3.micro" but the upstream `instanceType` attribute is
		// "t3.micro" — so the mapping uses `replace(host_instance_type, "mq.", "")`.
		"replace": strings.ReplaceAll,

		// strip_prefix(s, prefix) → removes a leading prefix if
		// present. Variant of replace() for the very common case of
		// "this string has a provider-side prefix we don't want".
		"strip_prefix": strings.TrimPrefix,

		// lower(s) → ASCII lowercase. Some upstream attribute values
		// arrive case-mismatched against Terraform's lowercase input;
		// authors use this on the catalog side rather than per
		// attribute in the parser.
		"lower": strings.ToLower,
	}
}

// EnvFor builds the variable environment for one resource evaluation.
// The PriceLookup closure captures the resource + mapping context so
// `price("name")` can resolve correctly at call time without the catalog
// or pricing source leaking into the expression scope.
//
// Attribute keys collide with stdlib function names at the user's risk:
// the function wins. Catalog authors who name an attribute `default` or
// `price` will be surprised, and that's acceptable.
func EnvFor(r domain.Resource, lookup PriceLookup, constants map[string]any) map[string]any {
	env := make(map[string]any, len(r.Attributes)+len(constants)+4)
	for k, v := range constants {
		env[k] = v
	}
	for k, v := range r.Attributes {
		env[k] = v
		// The Terraform parser surfaces nested blocks
		// (`root_block_device { volume_size = 50 }`) as nested maps,
		// but the catalog's attribute convention is flat snake_case
		// (`root_block_device_volume_size`). Expose one level of
		// flattened aliases so both shapes resolve; explicit
		// top-level attributes always win. Repeated blocks arrive
		// as []any — flatten the first occurrence, matching the
		// "price the primary block" convention.
		switch nested := v.(type) {
		case map[string]any:
			flattenInto(env, k, nested)
		case []any:
			if len(nested) > 0 {
				if m, ok := nested[0].(map[string]any); ok {
					flattenInto(env, k, m)
				}
			}
		}
	}
	for k, v := range stdFunctions(lookup) {
		env[k] = v
	}
	return env
}

// flattenInto adds prefix_subkey aliases for nested block levels,
// recursing up to two levels deep (covers Terraform's deepest common
// shape: `boot_disk { initialize_params { size = 100 } }` →
// `boot_disk_initialize_params_size`). Existing keys are never
// overwritten; repeated blocks alias their first occurrence.
func flattenInto(env map[string]any, prefix string, nested map[string]any) {
	flattenLevel(env, prefix, nested, 2)
}

func flattenLevel(env map[string]any, prefix string, nested map[string]any, depth int) {
	for sk, sv := range nested {
		key := prefix + "_" + sk
		if _, exists := env[key]; !exists {
			env[key] = sv
		}
		if depth <= 1 {
			continue
		}
		switch child := sv.(type) {
		case map[string]any:
			flattenLevel(env, key, child, depth-1)
		case []any:
			if len(child) > 0 {
				if m, ok := child[0].(map[string]any); ok {
					flattenLevel(env, key, m, depth-1)
				}
			}
		}
	}
}
