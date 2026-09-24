package terraform

import (
	"github.com/zclconf/go-cty/cty"
)

// collectLiteralResources exposes the statically known attributes of a
// module's resources, so `instance_type = aws_instance.base.instance_type`
// resolves when aws_instance.base sets instance_type to a literal.
//
// Only what is certain is exposed:
//   - attributes whose expression references nothing (literals, and
//     functions of literals); anything reading var, local, data, another
//     resource, count or each is left out, since its value could differ
//     from what the referencing expression expects;
//   - resources without count or for_each, whose address is a single
//     object (aws_instance.web[0] is not resolved).
//
// A reference to anything else (an attribute set from a variable, or one
// known only after apply, like `id`) fails to evaluate exactly as it did
// before, and the referencing attribute is reported unresolved.
//
// The result maps a resource type to an object of its resources, which
// moduleEnv adds to the evaluation scope under the type's name.
func collectLiteralResources(sources []sourceFile) map[string]cty.Value {
	byKind := map[string]map[string]cty.Value{}
	ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "resource" || len(block.Labels) < 2 {
				continue
			}
			kind, name := block.Labels[0], block.Labels[1]
			if reservedRoot(kind) {
				continue
			}
			if _, dup := byKind[kind][name]; dup {
				continue
			}
			if block.Body.Attributes["count"] != nil || block.Body.Attributes["for_each"] != nil {
				continue
			}
			attrs := map[string]cty.Value{}
			for _, attr := range block.Body.Attributes {
				switch attr.Name {
				case "depends_on", "provider", "lifecycle":
					continue
				}
				if len(attr.Expr.Variables()) > 0 {
					continue
				}
				val, diags := attr.Expr.Value(ctx)
				if diags.HasErrors() || !val.IsWhollyKnown() {
					continue
				}
				attrs[attr.Name] = val
			}
			if byKind[kind] == nil {
				byKind[kind] = map[string]cty.Value{}
			}
			byKind[kind][name] = cty.ObjectVal(attrs)
		}
	}
	out := make(map[string]cty.Value, len(byKind))
	for kind, names := range byKind {
		out[kind] = cty.ObjectVal(names)
	}
	return out
}

// reservedRoot reports names the evaluation scope already uses; a
// resource type can't legally take them, but a malformed configuration
// must not be able to shadow var or local.
func reservedRoot(name string) bool {
	switch name {
	case "var", "local", "data", "path", "count", "each", "module", "self", "terraform":
		return true
	}
	return false
}
