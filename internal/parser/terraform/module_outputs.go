package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// moduleOutputs evaluates a module instance's `output "x" { value = ... }`
// blocks in its own scope. An output that can't be evaluated is left out,
// so reading it fails in the caller exactly as an unknown attribute does.
func moduleOutputs(env moduleEnv, sources []sourceFile) cty.Value {
	ctx := env.evalContext(nil)
	out := map[string]cty.Value{}
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "output" || len(block.Labels) == 0 {
				continue
			}
			attr, ok := block.Body.Attributes["value"]
			if !ok {
				continue
			}
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				env.logger.Debug("module output evaluation failed",
					"output", block.Labels[0], "dir", env.scope.moduleDir,
					"diags", formatDiags(diags))
				continue
			}
			out[block.Labels[0]] = val
		}
	}
	return cty.ObjectVal(out)
}

// callOrder orders a configuration's module calls so that a call reading
// module.X, directly or through a local, comes after the call X. Calls
// that depend on nothing keep their source order. A cycle, which
// Terraform rejects unless it runs through outputs that don't depend on
// each other, is broken at the first remaining call in source order.
//
// localsReadModules reports whether any local reads a module output, in
// which case the locals are resolved again after each call.
func callOrder(calls []*moduleCall, sources []sourceFile) (order []int, localsReadModules bool) {
	localDeps := localModuleDeps(sources)
	for _, d := range localDeps {
		if len(d) > 0 {
			localsReadModules = true
			break
		}
	}
	names := map[string]bool{}
	deps := make([]map[string]bool, len(calls))
	for i, c := range calls {
		names[c.Labels[0]] = true
		deps[i] = map[string]bool{}
		for _, attr := range c.Body.Attributes {
			addModuleDeps(deps[i], attr.Expr.Variables(), localDeps)
		}
		delete(deps[i], c.Labels[0])
	}
	done := map[string]bool{}
	placed := make([]bool, len(calls))
	for len(order) < len(calls) {
		next := -1
		for i := range calls {
			if placed[i] {
				continue
			}
			if next < 0 {
				next = i
			}
			if ready(deps[i], names, done) {
				next = i
				break
			}
		}
		placed[next] = true
		order = append(order, next)
		done[calls[next].Labels[0]] = true
	}
	return order, localsReadModules
}

// ready reports whether every module call in deps has been resolved;
// names the configuration doesn't call can never be, so they don't wait.
func ready(deps, names, done map[string]bool) bool {
	for d := range deps {
		if names[d] && !done[d] {
			return false
		}
	}
	return true
}

// localModuleDeps maps each local to the module calls it reads, directly
// or through other locals.
func localModuleDeps(sources []sourceFile) map[string]map[string]bool {
	attrs := collectLocalAttributes(sources)
	direct := map[string]map[string]bool{}
	refs := map[string][]string{}
	for _, la := range attrs {
		direct[la.Name] = map[string]bool{}
		for _, t := range la.Expr.Variables() {
			switch t.RootName() {
			case "module":
				if name, ok := secondStep(t); ok {
					direct[la.Name][name] = true
				}
			case "local":
				if name, ok := secondStep(t); ok {
					refs[la.Name] = append(refs[la.Name], name)
				}
			}
		}
	}
	// Close over local → local references; chains are short.
	for changed := true; changed; {
		changed = false
		for name, deps := range direct {
			for _, r := range refs[name] {
				for d := range direct[r] {
					if !deps[d] {
						deps[d] = true
						changed = true
					}
				}
			}
		}
	}
	return direct
}

// addModuleDeps adds the module calls a set of traversals reads, directly
// or through locals, to deps.
func addModuleDeps(deps map[string]bool, traversals []hcl.Traversal, localDeps map[string]map[string]bool) {
	for _, t := range traversals {
		name, ok := secondStep(t)
		if !ok {
			continue
		}
		switch t.RootName() {
		case "module":
			deps[name] = true
		case "local":
			for d := range localDeps[name] {
				deps[d] = true
			}
		}
	}
}

// secondStep returns the attribute name after a traversal's root:
// "sizes" for module.sizes.x.
func secondStep(t hcl.Traversal) (string, bool) {
	if len(t) < 2 {
		return "", false
	}
	if a, ok := t[1].(hcl.TraverseAttr); ok {
		return a.Name, true
	}
	return "", false
}
