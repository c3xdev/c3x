package terraform

import (
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// providerRegions resolves the region a resource is billed in from the
// provider configuration that manages it.
//
// Before this existed the parser took the region of the first provider
// block it could evaluate and applied it to every resource, so any config
// with a second, aliased provider priced that provider's resources in the
// wrong region: a default us-east-1 provider plus an `aws.eu` alias priced
// both halves at us-east-1 rates.
type providerRegions struct {
	// fallback is the region used when nothing more specific resolves. It
	// is the old single-region answer, so configs that never use aliases
	// price exactly as before.
	fallback string
	// byKey maps a provider reference to its region: "aws" for the default
	// configuration, "aws.eu" for `alias = "eu"`.
	byKey map[string]string
	// byInstance holds OpenTofu's `for_each` providers: "aws.by_region" ->
	// instance key -> region, referenced as aws.by_region[each.key].
	byInstance map[string]map[string]string
}

// collectProviderRegions evaluates every provider block's region. vars,
// locals and data are the module's own; fallback is the region to use
// when a resource's provider cannot be resolved.
func collectProviderRegions(
	sources []sourceFile,
	vars, locals map[string]cty.Value,
	data cty.Value,
	fallback string,
	logger *slog.Logger,
) providerRegions {
	pr := providerRegions{
		fallback:   fallback,
		byKey:      map[string]string{},
		byInstance: map[string]map[string]string{},
	}
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "provider" || len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			attrName := providerRegionAttr(name)
			if attrName == "" {
				continue
			}
			key := name
			if alias, ok := evalString(block.Body.Attributes["alias"], vars, locals, data, nil); ok && alias != "" {
				key = name + "." + alias
			}

			// OpenTofu 1.9: one provider block, one configuration per
			// for_each element, each with its own region.
			if fe := block.Body.Attributes["for_each"]; fe != nil {
				val, diags := fe.Expr.Value(buildEvalContext(asObject(vars), asObject(locals), data, nil))
				if diags.HasErrors() {
					logger.Debug("provider for_each evaluation failed",
						"provider", key, "file", src.Path, "diags", formatDiags(diags))
					continue
				}
				instances := map[string]string{}
				for _, p := range foreachPairs(val) {
					each := map[string]cty.Value{"each": cty.ObjectVal(map[string]cty.Value{
						"key": cty.StringVal(p.Key), "value": p.Value,
					})}
					if r, ok := evalString(block.Body.Attributes[attrName], vars, locals, data, each); ok {
						instances[p.Key] = r
					}
				}
				pr.byInstance[key] = instances
				continue
			}

			if r, ok := evalString(block.Body.Attributes[attrName], vars, locals, data, nil); ok {
				if _, seen := pr.byKey[key]; !seen {
					pr.byKey[key] = r
				}
			}
		}
	}
	return pr
}

// forResource returns the region for one resource instance. ctx is the
// instance's evaluation context, so a `provider = aws.by_region[each.key]`
// reference resolves against that instance's each.key.
func (pr providerRegions) forResource(kind string, body *hclsyntax.Body, ctx *hcl.EvalContext) string {
	if attr := body.Attributes["provider"]; attr != nil {
		if r, ok := pr.fromReference(attr.Expr, ctx); ok {
			return r
		}
	}
	if r, ok := pr.byKey[providerForKind(kind)]; ok {
		return r
	}
	return pr.fallback
}

// fromReference resolves a `provider = ...` meta-argument: a plain
// reference (aws, aws.eu) or an indexed for_each instance
// (aws.by_region["use1"], aws.by_region[each.key]).
func (pr providerRegions) fromReference(expr hclsyntax.Expression, ctx *hcl.EvalContext) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		if key, ok := traversalKey(e.Traversal); ok {
			if len(e.Traversal) == 3 {
				// aws.by_region["use1"] parses as a traversal with an index step.
				if idx, ok := e.Traversal[2].(hcl.TraverseIndex); ok && idx.Key.Type() == cty.String {
					r, found := pr.byInstance[key][idx.Key.AsString()]
					return r, found
				}
			}
			r, found := pr.byKey[key]
			return r, found
		}
	case *hclsyntax.IndexExpr:
		coll, ok := e.Collection.(*hclsyntax.ScopeTraversalExpr)
		if !ok {
			return "", false
		}
		key, ok := traversalKey(coll.Traversal)
		if !ok {
			return "", false
		}
		v, diags := e.Key.Value(ctx)
		if diags.HasErrors() || v.IsNull() || !v.IsKnown() || v.Type() != cty.String {
			return "", false
		}
		r, found := pr.byInstance[key][v.AsString()]
		return r, found
	}
	return "", false
}

// forChild builds a child module's view: the child's own provider
// blocks win, then any explicit `providers = { aws = aws.eu }` mapping on
// the module block, then the parent's default configurations, which
// Terraform passes to a child implicitly.
func (pr providerRegions) forChild(child providerRegions, moduleBody *hclsyntax.Body, ctx *hcl.EvalContext) providerRegions {
	out := providerRegions{
		fallback:   child.fallback,
		byKey:      map[string]string{},
		byInstance: map[string]map[string]string{},
	}
	for k, v := range pr.byKey {
		if !strings.Contains(k, ".") {
			out.byKey[k] = v
		}
	}
	if attr := moduleBody.Attributes["providers"]; attr != nil {
		if obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
			for _, item := range obj.Items {
				childKey, ok := objectKeyReference(item.KeyExpr)
				if !ok {
					continue
				}
				if r, ok := pr.fromReference(item.ValueExpr, ctx); ok {
					out.byKey[childKey] = r
				}
			}
		}
	}
	for k, v := range child.byKey {
		out.byKey[k] = v
	}
	for k, v := range child.byInstance {
		out.byInstance[k] = v
	}
	if out.fallback == "" {
		out.fallback = pr.fallback
	}
	return out
}

// traversalKey turns aws or aws.eu (optionally followed by an index) into
// the map key used above.
func traversalKey(t hcl.Traversal) (string, bool) {
	if len(t) == 0 {
		return "", false
	}
	root, ok := t[0].(hcl.TraverseRoot)
	if !ok {
		return "", false
	}
	if len(t) >= 2 {
		if a, ok := t[1].(hcl.TraverseAttr); ok {
			return root.Name + "." + a.Name, true
		}
	}
	return root.Name, true
}

// objectKeyReference reads the left side of a `providers` map entry, which
// is a bare reference (aws, aws.west) rather than a string.
func objectKeyReference(expr hclsyntax.Expression) (string, bool) {
	if k, ok := expr.(*hclsyntax.ObjectConsKeyExpr); ok {
		expr = k.Wrapped
	}
	if st, ok := expr.(*hclsyntax.ScopeTraversalExpr); ok {
		return traversalKey(st.Traversal)
	}
	return "", false
}

// providerForKind maps a resource type to the local provider name that
// manages it by default: aws_instance -> aws, google_compute_instance ->
// google.
func providerForKind(kind string) string {
	name, _, _ := strings.Cut(kind, "_")
	return name
}

// evalString evaluates an optional attribute to a known string.
func evalString(attr *hclsyntax.Attribute, vars, locals map[string]cty.Value, data cty.Value, extras map[string]cty.Value) (string, bool) {
	if attr == nil {
		return "", false
	}
	v, diags := attr.Expr.Value(buildEvalContext(asObject(vars), asObject(locals), data, extras))
	if diags.HasErrors() || v.IsNull() || !v.IsKnown() || v.Type() != cty.String {
		return "", false
	}
	return v.AsString(), true
}
