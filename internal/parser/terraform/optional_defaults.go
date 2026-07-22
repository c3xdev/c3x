package terraform

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// applyOptionalDefaults inspects each variable block in the child module's
// sources and, for variables that have a `type` constraint containing
// `optional()` calls with default values, merges those defaults into the
// corresponding variable value in `vars`. This bridges the gap between
// Terraform's runtime type-system (which applies optional defaults
// automatically) and c3x's static HCL parsing.
//
// Example: given a variable typed as
//
//	variable "vm" {
//	  type = object({
//	    size     = optional(string, "Standard_D2s_v3")
//	    os_disk  = optional(object({
//	      disk_size_gb = optional(number, 64)
//	    }), {})
//	  })
//	}
//
// and a caller-supplied value of { dataDisks = [...] }, this function
// fills in os_disk = {} and, within os_disk, disk_size_gb = 64.
func applyOptionalDefaults(sources []sourceFile, vars map[string]cty.Value) {
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "variable" || len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			typeAttr, ok := block.Body.Attributes["type"]
			if !ok {
				continue
			}
			currentVal, hasVal := vars[name]
			if !hasVal || currentVal.IsNull() || !currentVal.IsKnown() {
				continue
			}

			// Extract optional() defaults from the type expression tree.
			defaults := extractOptionalDefaults(typeAttr.Expr)
			if len(defaults) == 0 {
				continue
			}

			// Merge defaults into the current value.
			merged := mergeDefaults(currentVal, defaults)
			if merged != cty.NilVal {
				vars[name] = merged
			}
		}
	}
}

// optionalDefault represents a single optional() default at one level
// of the type tree.
type optionalDefault struct {
	// defaultVal is the literal default value (2nd arg of optional()).
	defaultVal cty.Value
	// children holds nested optional() defaults for object members.
	children map[string]*optionalDefault
}

// extractOptionalDefaults walks an HCL expression tree representing a
// type constraint and finds all optional(type, default) calls. It
// returns a map of attribute-name → optionalDefault for the top-level
// object's members.
//
// The type constraint grammar we care about:
//
//	object({ key = optional(type, default), ... })
//
// We only process the subset we can statically understand.
func extractOptionalDefaults(expr hclsyntax.Expression) map[string]*optionalDefault {
	// The top-level expression should be a function call to object().
	call, ok := expr.(*hclsyntax.FunctionCallExpr)
	if !ok {
		return nil
	}
	if call.Name != "object" || len(call.Args) != 1 {
		return nil
	}
	// The argument to object() is an ObjectConsExpr: { key = type, ... }
	return extractObjectDefaults(call.Args[0])
}

// extractObjectDefaults processes the argument of an object() type
// function call, which is an ObjectConsExpr containing key-value pairs
// where values may be optional(type, default) calls.
func extractObjectDefaults(expr hclsyntax.Expression) map[string]*optionalDefault {
	obj, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil
	}
	out := make(map[string]*optionalDefault)
	for _, item := range obj.Items {
		// Key must be a traversal or literal string we can read.
		keyName := exprToLiteralString(item.KeyExpr)
		if keyName == "" {
			continue
		}
		// Value: check if it's optional(type, default)
		valCall, isCall := item.ValueExpr.(*hclsyntax.FunctionCallExpr)
		if !isCall || valCall.Name != "optional" {
			// Not optional — could still be a nested object() with
			// optional members inside it.
			if nestedCall, nestedOk := item.ValueExpr.(*hclsyntax.FunctionCallExpr); nestedOk && nestedCall.Name == "object" && len(nestedCall.Args) == 1 {
				children := extractObjectDefaults(nestedCall.Args[0])
				if len(children) > 0 {
					out[keyName] = &optionalDefault{children: children}
				}
			}
			continue
		}

		od := &optionalDefault{}

		// optional(type) — no default, skip
		// optional(type, default) — has default as 2nd arg
		if len(valCall.Args) >= 2 {
			// Try to evaluate the default expression as a literal.
			ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
			val, diags := valCall.Args[1].Value(ctx)
			if !diags.HasErrors() {
				od.defaultVal = val
			}
		}

		// The first arg of optional() is the type — check if it's
		// object({...}) so we can recurse for nested defaults.
		if len(valCall.Args) >= 1 {
			if innerCall, innerOk := valCall.Args[0].(*hclsyntax.FunctionCallExpr); innerOk && innerCall.Name == "object" && len(innerCall.Args) == 1 {
				od.children = extractObjectDefaults(innerCall.Args[0])
			}
		}

		if od.defaultVal != cty.NilVal || len(od.children) > 0 {
			out[keyName] = od
		}
	}
	return out
}

// exprToLiteralString tries to read a simple key expression as a plain
// string. Handles bare identifiers (the most common case in type
// constraints) and literal strings.
func exprToLiteralString(expr hclsyntax.Expression) string {
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		// ObjectConsKeyExpr wraps the actual key expression; recurse.
		return exprToLiteralString(e.Wrapped)
	case *hclsyntax.ScopeTraversalExpr:
		// Bare identifier like: os_disk = optional(...)
		if len(e.Traversal) == 1 {
			return e.Traversal.RootName()
		}
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			return e.Val.AsString()
		}
	}
	return ""
}

// mergeDefaults takes a cty.Value (the variable's current value) and
// fills in missing keys with their optional() defaults, recursing into
// nested objects.
func mergeDefaults(current cty.Value, defaults map[string]*optionalDefault) cty.Value {
	if !current.IsKnown() || current.IsNull() {
		return cty.NilVal
	}
	ty := current.Type()
	if !ty.IsObjectType() && !ty.IsMapType() {
		return cty.NilVal
	}

	// Convert to a mutable map of attribute values.
	attrs := make(map[string]cty.Value)
	it := current.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		attrs[k.AsString()] = v
	}

	changed := false
	for key, od := range defaults {
		existing, exists := attrs[key]

		if !exists || existing.IsNull() {
			// Key missing — apply default if we have one.
			if od.defaultVal != cty.NilVal {
				attrs[key] = applyChildDefaults(od.defaultVal, od.children)
				changed = true
			}
			// No explicit default: Terraform leaves the attribute null,
			// so we don't synthesize a value from child defaults alone.
			// Only optional(object({...}), {}) (with explicit 2nd arg)
			// should produce a populated object.
			continue
		}

		// Key exists — recurse into children if the value is an object.
		if len(od.children) > 0 && existing.IsKnown() && !existing.IsNull() {
			childTy := existing.Type()
			if childTy.IsObjectType() || childTy.IsMapType() {
				merged := mergeDefaults(existing, od.children)
				if merged != cty.NilVal {
					attrs[key] = merged
					changed = true
				}
			}
		}
	}

	if !changed {
		return cty.NilVal
	}
	return cty.ObjectVal(attrs)
}

// applyChildDefaults takes a default value and, if it's an object with
// child optional() defaults defined, recursively fills in those children.
// This handles the common pattern: `os_disk = optional(object({...}), {})`
// where the default is empty but the nested type has its own defaults.
func applyChildDefaults(defVal cty.Value, children map[string]*optionalDefault) cty.Value {
	if len(children) == 0 {
		return defVal
	}
	if !defVal.IsKnown() || defVal.IsNull() {
		return defVal
	}
	defTy := defVal.Type()
	if !defTy.IsObjectType() && !defTy.IsMapType() {
		return defVal
	}
	merged := mergeDefaults(defVal, children)
	if merged != cty.NilVal {
		return merged
	}
	return defVal
}
