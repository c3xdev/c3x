package terraform

import (
	"log/slog"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// resolveLocals evaluates every `locals { ... }` attribute against the
// growing scope until no more progress is made. Order of definition
// across files doesn't matter; locals that reference other locals
// resolve once the prerequisite has resolved in a prior pass.
//
// The fixed-point loop converges in O(depth) passes where depth is the
// longest local→local reference chain. In practice that's < 5 and the
// per-attribute work is cheap, so we don't bother with a smarter
// dependency-graph topological sort.
func resolveLocals(sources []sourceFile, vars map[string]cty.Value, data cty.Value, logger *slog.Logger) map[string]cty.Value {
	pending := collectLocalAttributes(sources)
	resolved := map[string]cty.Value{}
	for {
		progressed := false
		var still []localAttr
		for _, la := range pending {
			ctx := buildEvalContext(asObject(vars), asObject(resolved), data, nil)
			val, diags := la.Expr.Value(ctx)
			if diags.HasErrors() {
				still = append(still, la)
				continue
			}
			resolved[la.Name] = val
			progressed = true
		}
		pending = still
		if !progressed || len(pending) == 0 {
			break
		}
	}
	// Whatever never resolved: report it if the cause is a function we
	// don't implement, since every resource reading that local is affected.
	for _, la := range pending {
		_, diags := la.Expr.Value(buildEvalContext(asObject(vars), asObject(resolved), data, nil))
		warnUnknownFunctions(diags, logger, "local."+la.Name)
	}
	return resolved
}

// localAttr pairs a local's name with its HCL expression so the
// fixed-point loop can defer evaluation without re-walking blocks.
type localAttr struct {
	Name string
	Expr hclsyntax.Expression
}

func collectLocalAttributes(sources []sourceFile) []localAttr {
	var out []localAttr
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "locals" {
				continue
			}
			for _, attr := range block.Body.Attributes {
				out = append(out, localAttr{Name: attr.Name, Expr: attr.Expr})
			}
		}
	}
	return out
}
