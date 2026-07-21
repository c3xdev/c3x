package terraform

import (
	"fmt"
	"log/slog"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// emitResources walks every `resource "kind" "name" { ... }` block and
// produces one or more domain.Resource entries depending on the block's
// `count` / `for_each` meta-arguments.
//
// Region defaulting: each emitted Resource carries the
// provider-detected region. The calculator falls back to its own
// configured default when the Resource has none, so we don't pad here.
func emitResources(
	sources []sourceFile,
	vars map[string]cty.Value,
	locals map[string]cty.Value,
	data cty.Value,
	region string,
	logger *slog.Logger,
) ([]domain.Resource, error) {
	var out []domain.Resource
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "resource" || len(block.Labels) < 2 {
				continue
			}
			kind := block.Labels[0]
			name := block.Labels[1]
			if err := emitOne(src.Path, kind, name, block.Body, vars, locals, data, region, "", logger, &out); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// emitOne handles the count/for_each expansion for a single resource
// block and pushes domain.Resources onto `out`. namePrefix is the
// module-path prefix (e.g. `module.vpc.`); empty for top-level blocks.
func emitOne(
	srcPath, kind, name string,
	body *hclsyntax.Body,
	vars, locals map[string]cty.Value,
	data cty.Value,
	region string,
	namePrefix string,
	logger *slog.Logger,
	out *[]domain.Resource,
) error {
	countAttr := body.Attributes["count"]
	foreachAttr := body.Attributes["for_each"]

	switch {
	case countAttr != nil:
		baseCtx := buildEvalContext(asObject(vars), asObject(locals), data, nil)
		val, diags := countAttr.Expr.Value(baseCtx)
		if diags.HasErrors() {
			logger.Warn("count evaluation failed; resource omitted",
				"file", srcPath, "resource", kind+"."+name,
				"diags", formatDiags(diags))
			return nil
		}
		count, ok := readCount(val, logger, srcPath, kind, name)
		if !ok {
			return nil
		}
		for i := 0; i < count; i++ {
			extras := map[string]cty.Value{
				"count": cty.ObjectVal(map[string]cty.Value{
					"index": cty.NumberIntVal(int64(i)),
				}),
			}
			ctx := buildEvalContext(asObject(vars), asObject(locals), data, extras)
			attrs, err := extractAttributes(body, ctx)
			if err != nil {
				return fmt.Errorf("%s.%s[%d]: %w", kind, name, i, err)
			}
			*out = append(*out, makeResource(kind, fmt.Sprintf("%s%s[%d]", namePrefix, name, i), attrs, region))
		}
		return nil

	case foreachAttr != nil:
		baseCtx := buildEvalContext(asObject(vars), asObject(locals), data, nil)
		val, diags := foreachAttr.Expr.Value(baseCtx)
		if diags.HasErrors() {
			logger.Warn("for_each evaluation failed; resource omitted",
				"file", srcPath, "resource", kind+"."+name,
				"diags", formatDiags(diags))
			return nil
		}
		pairs := foreachPairs(val)
		for _, p := range pairs {
			extras := map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"key":   cty.StringVal(p.Key),
					"value": p.Value,
				}),
			}
			ctx := buildEvalContext(asObject(vars), asObject(locals), data, extras)
			attrs, err := extractAttributes(body, ctx)
			if err != nil {
				return fmt.Errorf("%s.%s[%q]: %w", kind, name, p.Key, err)
			}
			*out = append(*out, makeResource(
				kind,
				fmt.Sprintf("%s%s[%q]", namePrefix, name, p.Key),
				attrs, region,
			))
		}
		return nil
	}

	ctx := buildEvalContext(asObject(vars), asObject(locals), data, nil)
	attrs, err := extractAttributes(body, ctx)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", kind, name, err)
	}
	*out = append(*out, makeResource(kind, namePrefix+name, attrs, region))
	return nil
}

// readCount narrows an evaluated `count` value to a non-negative int.
// Booleans map to 0/1 (the conditional pattern); anything else logs a
// warning and yields 0 so the resource is omitted.
func readCount(v cty.Value, logger *slog.Logger, srcPath, kind, name string) (int, bool) {
	if v.IsNull() {
		return 0, true
	}
	switch v.Type() {
	case cty.Number:
		bf := v.AsBigFloat()
		n, _ := bf.Int64()
		if n < 0 {
			return 0, true
		}
		return int(n), true
	case cty.Bool:
		if v.True() {
			return 1, true
		}
		return 0, true
	}
	logger.Warn("count expression returned an unsupported type",
		"file", srcPath, "resource", kind+"."+name, "type", v.Type().FriendlyName())
	return 0, false
}

// foreachPair is one (key, value) pair produced by expanding for_each.
type foreachPair struct {
	Key   string
	Value cty.Value
}

func foreachPairs(v cty.Value) []foreachPair {
	if !v.IsKnown() || v.IsNull() {
		return nil
	}
	ty := v.Type()
	var out []foreachPair
	switch {
	case ty.IsMapType(), ty.IsObjectType():
		it := v.ElementIterator()
		for it.Next() {
			k, elem := it.Element()
			out = append(out, foreachPair{Key: k.AsString(), Value: elem})
		}
	case ty.IsSetType(), ty.IsListType(), ty.IsTupleType():
		it := v.ElementIterator()
		for it.Next() {
			_, elem := it.Element()
			var key string
			if elem.Type() == cty.String {
				key = elem.AsString()
			} else {
				key = elem.GoString()
			}
			out = append(out, foreachPair{Key: key, Value: elem})
		}
	}
	return out
}

// extractAttributes walks a resource body and produces the flat
// attribute map the calculator reads. Meta-arguments are filtered;
// nested blocks (like `root_block_device { volume_size = 50 }` on
// aws_instance) become nested map[string]any entries so catalog
// expressions can reach them via `root_block_device.volume_size`.
func extractAttributes(body *hclsyntax.Body, ctx *hcl.EvalContext) (map[string]any, error) {
	return extractAttributesLevel(body, ctx, true)
}

func extractAttributesLevel(body *hclsyntax.Body, ctx *hcl.EvalContext, topLevel bool) (map[string]any, error) {
	out := map[string]any{}
	for _, attr := range body.Attributes {
		// Meta-arguments are Terraform syntax, not resource data —
		// but ONLY at the resource's top level. Inside nested blocks
		// the same names are real arguments (`guest_accelerator {
		// count = 1 }` is a GPU count, not a resource replicator).
		if topLevel {
			switch attr.Name {
			case "count", "for_each", "depends_on", "provider", "lifecycle":
				continue
			}
		}
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			// Attribute couldn't be resolved (e.g. optional() defaults
			// in module variables that aren't supplied by the caller).
			// Store nil so catalog expressions' `default(x, fallback)`
			// correctly falls through to the fallback value. Storing a
			// non-nil placeholder (like the source range string) would
			// bypass the default() logic and trigger type-mismatch
			// errors in numeric comparisons.
			out[attr.Name] = nil
			continue
		}
		out[attr.Name] = ctyToAny(val)
	}
	for _, block := range body.Blocks {
		if block.Type == "locals" {
			continue
		}
		nested, err := extractAttributesLevel(block.Body, ctx, false)
		if err != nil {
			return nil, err
		}
		key := block.Type
		switch existing := out[key].(type) {
		case nil:
			out[key] = nested
		case []any:
			out[key] = append(existing, nested)
		default:
			out[key] = []any{existing, nested}
		}
	}
	return out, nil
}

func makeResource(kind, name string, attrs map[string]any, region string) domain.Resource {
	r := domain.Resource{
		Ref:        domain.Reference{Kind: kind, Name: name},
		Attributes: attrs,
	}
	if region != "" {
		r.Region = &region
	}
	return r
}
