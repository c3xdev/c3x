package terraform

import (
	"fmt"
	"log/slog"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// moduleEnv is everything the expressions of one module instance are
// evaluated against: its variables, locals, data placeholders, the
// literal attributes of its resources, and its provider regions.
type moduleEnv struct {
	scope  evalScope
	vars   map[string]cty.Value
	locals map[string]cty.Value
	data   cty.Value
	// resources exposes the statically known attributes of the module's
	// resources as <type>.<name>.<attr>; see collectLiteralResources.
	resources map[string]cty.Value
	regions   providerRegions
	logger    *slog.Logger
}

// evalContext builds the context for one expression. extras are the
// per-instance scopes (count, each).
func (e moduleEnv) evalContext(extras map[string]cty.Value) *hcl.EvalContext {
	ctx := e.scope.evalContext(asObject(e.vars), asObject(e.locals), e.data, extras)
	for k, v := range e.resources {
		if _, taken := ctx.Variables[k]; !taken {
			ctx.Variables[k] = v
		}
	}
	return ctx
}

// emitResources walks every `resource "kind" "name" { ... }` block and
// produces one or more domain.Resource entries depending on the block's
// `count` / `for_each` meta-arguments.
//
// Region: each emitted Resource carries the region of the provider that
// manages it (see providerRegions). The calculator falls back to its own
// configured default when the Resource has none, so we don't pad here.
func emitResources(env moduleEnv, sources []sourceFile, namePrefix string, out *[]domain.Resource) error {
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "resource" || len(block.Labels) < 2 {
				continue
			}
			kind := block.Labels[0]
			name := block.Labels[1]
			if err := emitOne(env, src.Path, kind, name, block.Body, namePrefix, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// emitOne handles the count/for_each expansion for a single resource
// block and pushes domain.Resources onto `out`. namePrefix is the
// module-path prefix (e.g. `module.vpc.`); empty for top-level blocks.
func emitOne(
	env moduleEnv,
	srcPath, kind, name string,
	body *hclsyntax.Body,
	namePrefix string,
	out *[]domain.Resource,
) error {
	address := namePrefix + kind + "." + name
	instances, ok := expandInstances(env, body, srcPath, address)
	if !ok {
		return nil
	}
	if err := env.scope.budget.instances(address, len(instances)); err != nil {
		return err
	}
	for _, inst := range instances {
		ctx := env.evalContext(inst.extras)
		attrs, unresolved, err := extractAttributes(body, ctx, env.scope.budget, env.logger, address+inst.suffix)
		if err != nil {
			return fmt.Errorf("%s.%s%s: %w", kind, name, inst.suffix, err)
		}
		*out = append(*out, makeResource(kind, namePrefix+name+inst.suffix, attrs, env.regions.forResource(kind, body, ctx), unresolved))
	}
	return nil
}

// instance is one expansion of a block's count / for_each: the address
// suffix ("", "[0]", `["a"]`) and the count / each scope it evaluates in.
type instance struct {
	suffix string
	extras map[string]cty.Value
}

// expandInstances evaluates a resource or module block's count /
// for_each. ok is false when the meta-argument can't be evaluated, in
// which case the block is omitted (with a warning). A count or for_each
// computed from a data source placeholder is expanded, and warned about.
func expandInstances(env moduleEnv, body *hclsyntax.Body, srcPath, address string) ([]instance, bool) {
	countAttr := body.Attributes["count"]
	foreachAttr := body.Attributes["for_each"]
	logger := env.logger

	switch {
	case countAttr != nil:
		val, diags := countAttr.Expr.Value(env.evalContext(nil))
		if diags.HasErrors() {
			logger.Warn("count evaluation failed; block omitted",
				"file", srcPath, "resource", address,
				"diags", formatDiags(diags))
			return nil, false
		}
		val, deps := stripPlaceholders(val)
		count, ok := readCount(val, logger, srcPath, address)
		if !ok {
			return nil, false
		}
		if len(deps) > 0 {
			warnPlaceholders(logger, "count", address, deps)
		}
		// Bounded before allocating: count = 1e9 must not build a slice.
		if count > maxInstancesPerResource {
			count = maxInstancesPerResource + 1
		}
		out := make([]instance, count)
		for i := range out {
			out[i] = instance{
				suffix: fmt.Sprintf("[%d]", i),
				extras: map[string]cty.Value{
					"count": cty.ObjectVal(map[string]cty.Value{"index": cty.NumberIntVal(int64(i))}),
				},
			}
		}
		return out, true

	case foreachAttr != nil:
		val, diags := foreachAttr.Expr.Value(env.evalContext(nil))
		if diags.HasErrors() {
			logger.Warn("for_each evaluation failed; block omitted",
				"file", srcPath, "resource", address,
				"diags", formatDiags(diags))
			return nil, false
		}
		val, deps := stripPlaceholders(val)
		if len(deps) > 0 {
			warnPlaceholders(logger, "for_each", address, deps)
		}
		pairs := foreachPairs(val)
		out := make([]instance, len(pairs))
		for i, p := range pairs {
			// Every instance of a for_each built from a placeholder is
			// itself a guess: carry the marks so attributes computed
			// from each.key / each.value are reported unresolved.
			out[i] = instance{
				suffix: fmt.Sprintf("[%q]", p.Key),
				extras: map[string]cty.Value{
					"each": cty.ObjectVal(map[string]cty.Value{
						"key":   withPlaceholders(cty.StringVal(p.Key), deps),
						"value": withPlaceholders(p.Value, deps),
					}),
				},
			}
		}
		return out, true
	}
	return []instance{{}}, true
}

// readCount narrows an evaluated `count` value to a non-negative int.
// Booleans map to 0/1 (the conditional pattern); anything else logs a
// warning and yields 0 so the resource is omitted.
func readCount(v cty.Value, logger *slog.Logger, srcPath, address string) (int, bool) {
	if v.IsNull() || !v.IsKnown() {
		return 0, true
	}
	switch v.Type() {
	case cty.Number:
		bf := v.AsBigFloat()
		n, _ := bf.Int64()
		if n < 0 {
			return 0, true
		}
		if n > int64(maxInstancesPerResource) {
			return maxInstancesPerResource + 1, true
		}
		return int(n), true
	case cty.Bool:
		if v.True() {
			return 1, true
		}
		return 0, true
	}
	logger.Warn("count expression returned an unsupported type",
		"file", srcPath, "resource", address, "type", v.Type().FriendlyName())
	return 0, false
}

// foreachPair is one (key, value) pair produced by expanding for_each.
type foreachPair struct {
	Key   string
	Value cty.Value
}

// foreachPairs lists a for_each collection's instances. v must be
// unmarked (see stripPlaceholders).
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
			if elem.Type() == cty.String && elem.IsKnown() && !elem.IsNull() {
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
// `dynamic` blocks are expanded into the same shape (see expandDynamic).
func extractAttributes(body *hclsyntax.Body, ctx *hcl.EvalContext, budget *parseBudget, logger *slog.Logger, where string) (map[string]any, []string, error) {
	x := attrExtractor{budget: budget, logger: logger}
	attrs, err := x.level(body, ctx, true, where, "")
	return attrs, x.unresolved, err
}

type attrExtractor struct {
	budget     *parseBudget
	logger     *slog.Logger
	unresolved []string
}

func (x *attrExtractor) level(body *hclsyntax.Body, ctx *hcl.EvalContext, topLevel bool, where, path string) (map[string]any, error) {
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
			warnUnknownFunctions(diags, x.logger, where+"."+attr.Name)
			x.unresolved = append(x.unresolved, path+attr.Name)
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
		val, deps := stripPlaceholders(val)
		if len(deps) > 0 {
			// Computed from a data source c3x only has a placeholder
			// for: as unknown as before placeholders existed.
			x.logger.Debug("attribute depends on a data source placeholder; left unresolved",
				"attribute", where+"."+attr.Name, "data", deps[0].Ref)
			x.unresolved = append(x.unresolved, path+attr.Name)
			out[attr.Name] = nil
			continue
		}
		out[attr.Name] = ctyToAny(val)
	}
	for _, block := range body.Blocks {
		switch block.Type {
		case "locals":
			continue
		case "dynamic":
			if err := x.dynamic(block, ctx, where, path, out); err != nil {
				return nil, err
			}
			continue
		}
		nested, err := x.level(block.Body, ctx, false, where+"."+block.Type, path+block.Type+".")
		if err != nil {
			return nil, err
		}
		addBlock(out, block.Type, nested)
	}
	return out, nil
}

// addBlock records one nested block under its type. One block is a map;
// a repeated block is a list of maps, in source order. The catalog's
// sum_field / count_blocks read both shapes.
func addBlock(out map[string]any, key string, nested map[string]any) {
	switch existing := out[key].(type) {
	case nil:
		out[key] = nested
	case []any:
		out[key] = append(existing, nested)
	default:
		out[key] = []any{existing, nested}
	}
}

// dynamic expands a `dynamic "label" { for_each = ...; content { ... } }`
// block the way Terraform does: one `label` block per element of
// for_each, each evaluated with the iterator (named by `iterator`,
// defaulting to the label) bound to {key, value}. The result has exactly
// the shape of the equivalent literal blocks, so catalogs can't tell
// them apart. Nested dynamic blocks inside content expand recursively
// with the outer iterator still in scope.
func (x *attrExtractor) dynamic(block *hclsyntax.Block, ctx *hcl.EvalContext, where, path string, out map[string]any) error {
	if len(block.Labels) != 1 {
		return nil
	}
	label := block.Labels[0]
	var content *hclsyntax.Block
	for _, b := range block.Body.Blocks {
		if b.Type == "content" {
			content = b
			break
		}
	}
	feAttr := block.Body.Attributes["for_each"]
	if content == nil || feAttr == nil {
		return nil
	}
	iterName := label
	if it := block.Body.Attributes["iterator"]; it != nil {
		name := hcl.ExprAsKeyword(it.Expr)
		if name == "" {
			x.logger.Debug("dynamic block iterator is not a name; block skipped",
				"block", where+"."+label)
			return nil
		}
		iterName = name
	}
	coll, diags := feAttr.Expr.Value(ctx)
	if diags.HasErrors() {
		warnUnknownFunctions(diags, x.logger, where+"."+label+".for_each")
		x.logger.Debug("dynamic block for_each could not be evaluated; blocks omitted",
			"block", where+"."+label, "diags", formatDiags(diags))
		x.unresolved = append(x.unresolved, path+label)
		return nil
	}
	coll, deps := stripPlaceholders(coll)
	if len(deps) > 0 {
		warnPlaceholders(x.logger, "dynamic block for_each", where+"."+label, deps)
	}
	if coll.IsNull() || !coll.IsKnown() || !coll.CanIterateElements() {
		return nil
	}
	if err := x.budget.dynamicBlocks(where+"."+label, coll.LengthInt()); err != nil {
		return err
	}
	for it := coll.ElementIterator(); it.Next(); {
		k, v := it.Element()
		child := ctx.NewChild()
		child.Variables = map[string]cty.Value{
			iterName: cty.ObjectVal(map[string]cty.Value{
				"key":   withPlaceholders(k, deps),
				"value": withPlaceholders(v, deps),
			}),
		}
		nested, err := x.level(content.Body, child, false, where+"."+label, path+label+".")
		if err != nil {
			return err
		}
		addBlock(out, label, nested)
	}
	return nil
}

func makeResource(kind, name string, attrs map[string]any, region string, unresolved []string) domain.Resource {
	r := domain.Resource{
		Ref:        domain.Reference{Kind: kind, Name: name},
		Attributes: attrs,
		Unresolved: unresolved,
	}
	if region != "" {
		r.Region = &region
	}
	return r
}
