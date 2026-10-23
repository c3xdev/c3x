package terraform

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// MaxModuleDepth caps how deep we'll recurse into nested modules.
// Real configs rarely nest past 2–3; this exists purely to catch
// self-referential `module "x" { source = "." }` cycles.
const MaxModuleDepth = 10

// loadInitModules reads `<dir>/.terraform/modules/modules.json` if it
// exists — that's the manifest Terraform writes after `terraform init`,
// mapping each module key (`vpc`, `vpc.subnet`, …) to a resolved local
// directory. With it present, we transparently support registry / git
// modules by descending into the manifest-resolved paths.
func loadInitModules(baseDir string, logger *slog.Logger) map[string]string {
	manifest := filepath.Join(baseDir, ".terraform", "modules", "modules.json")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return nil
	}
	var data struct {
		Modules []struct {
			Key string `json:"Key"`
			Dir string `json:"Dir"`
		} `json:"Modules"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		logger.Warn("failed to parse modules.json; ignoring registry modules",
			"path", manifest, "error", err)
		return nil
	}
	out := map[string]string{}
	for _, m := range data.Modules {
		if m.Key == "" {
			continue
		}
		out[m.Key] = filepath.Join(baseDir, m.Dir)
	}
	return out
}

// moduleExpander carries what stays fixed while resolving a module tree.
type moduleExpander struct {
	initModules map[string]string
	offline     bool
	logger      *slog.Logger
}

// moduleNode is one module instance: the scope its expressions evaluate
// in, its configuration, the address prefix of its resources (empty for
// the root, `module.x[0].` below it), and the module instances it calls,
// in source order.
type moduleNode struct {
	env      moduleEnv
	sources  []sourceFile
	prefix   string
	children []*moduleNode
}

// emit appends the resources of n and of every module instance under it.
func (n *moduleNode) emit(out *[]domain.Resource) error {
	if err := emitResources(n.env, n.sources, n.prefix, out); err != nil {
		return err
	}
	for _, c := range n.children {
		if err := c.emit(out); err != nil {
			return err
		}
	}
	return nil
}

// resolveCalls resolves every `module "X" { ... }` block of n: it parses
// each module's source directory, threading n's values in as the child's
// variables, recurses into the child's own module calls, and evaluates
// the child's `output` blocks. Each call's outputs are added to n's scope
// as module.X, and n's locals are resolved again, before the next call is
// resolved, so a module input, a local or a resource of n can read
// another module's outputs. Calls are resolved in dependency order (see
// callOrder); n's locals are resolved a final time once all are done.
//
// A module block with count or for_each is resolved once per instance,
// with count.index / each.key / each.value in scope for its inputs, and
// its resources are addressed module.X[0]. / module.X["a"]., as Terraform
// addresses them. Every instance counts against the parse budget.
//
// Local sources resolve via the filesystem; registry/git sources use
// the manifest produced by `terraform init`. Sources we can't resolve
// (no manifest entry) log a loud warning and are skipped — the rest of
// the config still produces a useful estimate.
func (m moduleExpander) resolveCalls(n *moduleNode, baseDir, keyPrefix string, depth int) error {
	calls := moduleBlocks(n.sources)
	if len(calls) > 0 && depth >= MaxModuleDepth {
		m.logger.Warn("module nesting exceeded MaxModuleDepth; truncating expansion. "+
			"likely a self-referential `module \"x\" { source = \".\" }`",
			"depth", depth, "max", MaxModuleDepth)
		calls = nil
	}
	order, localsReadModules := callOrder(calls, n.sources)
	byCall := make([][]*moduleNode, len(calls))
	for _, i := range order {
		nodes, outputs, ok, err := m.resolveCall(n, calls[i], baseDir, keyPrefix, depth)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		byCall[i] = nodes
		if n.env.modules == nil {
			n.env.modules = map[string]cty.Value{}
		}
		n.env.modules[calls[i].Labels[0]] = outputs
		if localsReadModules {
			n.env.locals = resolveLocals(n.env.scope, n.sources, n.env.vars, n.env.data, asObject(n.env.modules), nil)
		}
	}
	n.env.locals = resolveLocals(n.env.scope, n.sources, n.env.vars, n.env.data, asObject(n.env.modules), m.logger)
	for _, nodes := range byCall {
		n.children = append(n.children, nodes...)
	}
	return nil
}

// moduleBlocks lists the module calls of a configuration.
func moduleBlocks(sources []sourceFile) []*moduleCall {
	var out []*moduleCall
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type == "module" && len(block.Labels) > 0 {
				out = append(out, &moduleCall{Block: block, path: src.Path})
			}
		}
	}
	return out
}

// moduleCall is a module block and the file it is in.
type moduleCall struct {
	*hclsyntax.Block
	path string
}

// resolveCall resolves one module call into its instances, and returns
// the value module.<name> takes in the caller: the outputs object, a
// tuple of them under count, or an object keyed by each.key under
// for_each. ok is false when the call is skipped.
func (m moduleExpander) resolveCall(
	parent *moduleNode,
	block *moduleCall,
	baseDir, keyPrefix string,
	depth int,
) (nodes []*moduleNode, outputs cty.Value, ok bool, err error) {
	logger := m.logger
	env := parent.env
	modName := block.Labels[0]
	address := parent.prefix + "module." + modName

	instances, ok := expandInstances(env, block.Body, block.path, address)
	if !ok {
		return nil, cty.NilVal, false, nil
	}
	if err := env.scope.budget.moduleCall(address, len(instances)); err != nil {
		return nil, cty.NilVal, false, err
	}
	_, counted := block.Body.Attributes["count"]
	_, keyed := block.Body.Attributes["for_each"]
	if len(instances) == 0 {
		if counted {
			return nil, cty.EmptyTupleVal, true, nil
		}
		return nil, cty.EmptyObjectVal, true, nil
	}

	parentCtx := env.evalContext(nil)
	source, ok := readModuleSource(block.Body, parentCtx)
	if !ok {
		return nil, cty.NilVal, false, nil
	}
	version := readModuleVersion(block.Body, parentCtx)

	// The init manifest keys modules by name, never by instance.
	manifestKey := modName
	if keyPrefix != "" {
		manifestKey = keyPrefix + "." + modName
	}

	childDir, ok := resolveModuleSource(source, version, baseDir, manifestKey, m.initModules, modName, m.offline, logger)
	if !ok {
		return nil, cty.NilVal, false, nil
	}
	if info, err := os.Stat(childDir); err != nil || !info.IsDir() {
		logger.Warn("module source path is not a directory; skipping",
			"module", modName, "dir", childDir)
		return nil, cty.NilVal, false, nil
	}
	// Untrusted input may not pull configuration from outside its own
	// tree: on a server, source = "../other-upload" (or a committed
	// .terraform/modules/modules.json pointing there) would fold
	// another user's infrastructure into this estimate. Terraform
	// allows ../ sources, so trusted parses keep doing the same.
	if env.scope.untrusted && !env.scope.containsDir(childDir) {
		logger.Warn("untrusted input: module source is outside the project directory; skipping",
			"module", modName, "dir", childDir)
		return nil, cty.NilVal, false, nil
	}

	childSources, err := loadConfigDir(childDir, logger)
	if err != nil {
		return nil, cty.NilVal, false, fmt.Errorf("module %s: %w", modName, err)
	}
	child := moduleSource{
		dir:       childDir,
		sources:   childSources,
		types:     collectVariableTypes(childSources),
		resources: collectLiteralResources(childSources),
	}
	var tuple []cty.Value
	byKey := map[string]cty.Value{}
	for _, inst := range instances {
		node, err := m.resolveInstance(env, block.Block, child, inst, address+inst.suffix+".", manifestKey, depth)
		if err != nil {
			return nil, cty.NilVal, false, err
		}
		nodes = append(nodes, node)
		out := moduleOutputs(node.env, child.sources)
		tuple = append(tuple, out)
		byKey[inst.key] = out
	}
	switch {
	case counted:
		return nodes, cty.TupleVal(tuple), true, nil
	case keyed:
		return nodes, cty.ObjectVal(byKey), true, nil
	}
	return nodes, tuple[0], true, nil
}

// moduleSource is a module's configuration, loaded once per module block
// and shared by every instance of it.
type moduleSource struct {
	dir       string
	sources   []sourceFile
	types     map[string]variableTypeConstraint
	resources map[string]cty.Value
}

// resolveInstance resolves one instance of a module call: its inputs are
// evaluated in the parent's scope plus the instance's count / each, then
// the child's own module calls are resolved in turn. Its resources are
// emitted later, under childPrefix (see moduleNode.emit).
func (m moduleExpander) resolveInstance(
	parent moduleEnv,
	block *hclsyntax.Block,
	child moduleSource,
	inst instance,
	childPrefix, manifestKey string,
	depth int,
) (*moduleNode, error) {
	logger := m.logger
	modName := block.Labels[0]
	instCtx := parent.evalContext(inst.extras)

	// Evaluate every non-meta attribute against the parent's
	// scope to build the child's var.x inputs.
	childInputs := map[string]cty.Value{}
	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "source", "version", "count", "for_each", "providers", "depends_on":
			continue
		}
		val, diags := attr.Expr.Value(instCtx)
		if diags.HasErrors() {
			logger.Debug("module input evaluation failed",
				"module", modName, "input", attr.Name,
				"diags", formatDiags(diags))
			continue
		}
		childInputs[attr.Name] = val
	}

	childVars, err := collectVariableDefaults(child.sources)
	if err != nil {
		return nil, err
	}
	if err := applyAutoTfvars(child.dir, childVars); err != nil {
		return nil, err
	}
	for k, v := range childInputs {
		childVars[k] = v
	}
	// Normalise the child's inputs (and defaults) to the child's
	// declared `type` constraints, filling optional() attribute
	// defaults. This bridges Terraform's runtime type-system and
	// c3x's static parsing: it applies explicit `optional(t, def)`
	// defaults and materialises bare `optional(t)` as null, so a
	// caller value like `instances = { one = {} }` exposes
	// `each.value.instance_class` instead of erroring inside the
	// module's expressions.
	applyVariableTypes(childVars, child.types)

	childFallback := parent.regions.fallback
	regionHints := parent.regions.forChild(
		collectProviderRegions(child.sources, childVars, nil, cty.EmptyObjectVal, childFallback, logger),
		block.Body, instCtx,
	)
	childData := collectDataBlocks(child.sources, regionHints)
	childScope := parent.scope.child(child.dir)
	childLocals := resolveLocals(childScope, child.sources, childVars, childData, cty.EmptyObjectVal, nil)
	if r := findDefaultRegion(child.sources, childVars, childLocals, childData, logger); r != "" {
		childFallback = r
	}
	childRegions := parent.regions.forChild(
		collectProviderRegions(child.sources, childVars, childLocals, childData, childFallback, logger),
		block.Body, instCtx,
	)

	env := moduleEnv{
		scope:     childScope,
		vars:      childVars,
		locals:    childLocals,
		data:      childData,
		resources: child.resources,
		regions:   childRegions,
		logger:    logger,
		// A module instance that exists because of a placeholder
		// count / for_each makes every resource under it a guess too.
		assumedCount: joinAssumed(parent.assumedCount, inst.assumed),
	}
	node := &moduleNode{env: env, sources: child.sources, prefix: childPrefix}
	if err := m.resolveCalls(node, child.dir, manifestKey, depth+1); err != nil {
		return nil, err
	}
	return node, nil
}

// readModuleSource evaluates a module block's `source = "..."` attribute
// to a string. Non-string sources (data refs, computed paths) are
// skipped silently since they can't drive a filesystem lookup.
func readModuleSource(body *hclsyntax.Body, ctx *hcl.EvalContext) (string, bool) {
	return readModuleStringAttr(body, ctx, "source")
}

// readModuleVersion reads the module block's `version` constraint
// ("5.1.0", "~> 3.0", ...). Empty when unset — registry fetches
// resolve to latest in that case.
func readModuleVersion(body *hclsyntax.Body, ctx *hcl.EvalContext) string {
	v, _ := readModuleStringAttr(body, ctx, "version")
	return v
}

func readModuleStringAttr(body *hclsyntax.Body, ctx *hcl.EvalContext, name string) (string, bool) {
	attr, ok := body.Attributes[name]
	if !ok {
		return "", false
	}
	val, diags := attr.Expr.Value(ctx)
	val, _ = stripPlaceholders(val)
	if diags.HasErrors() || val.Type() != cty.String || val.IsNull() || !val.IsKnown() {
		return "", false
	}
	return val.AsString(), true
}

// moduleFetcher is the package-level cache of remote-module fetches.
// We construct it lazily on first use so callers that never hit a
// remote module pay zero cost.
//
// The variable is package-private and mutex-guarded so concurrent
// parsers share the same on-disk cache without racing.
var (
	moduleFetcher     *ModuleFetcher
	moduleFetcherOnce sync.Once
	moduleFetcherErr  error
)

func sharedModuleFetcher() (*ModuleFetcher, error) {
	moduleFetcherOnce.Do(func() {
		moduleFetcher, moduleFetcherErr = NewModuleFetcher("")
	})
	return moduleFetcher, moduleFetcherErr
}

func resolveModuleSource(
	source, version, baseDir, manifestKey string,
	initModules map[string]string,
	modName string,
	offline bool,
	logger *slog.Logger,
) (string, bool) {
	if isLocalSource(source) {
		return filepath.Join(baseDir, source), true
	}
	// .terraform/modules/modules.json wins when present — Terraform's
	// own resolver has already laid the module out for us.
	if resolved, ok := initModules[manifestKey]; ok {
		return resolved, true
	}
	// Native fetcher: Terraform Registry, Git, HTTP archive.
	// Gated on the offline flag: `--offline` estimates (and every
	// test) must never reach the network for module resolution.
	if !offline && detectSourceType(source) != sourceUnknown {
		f, err := sharedModuleFetcher()
		if err != nil {
			logger.Warn("module fetcher unavailable",
				"module", modName, "source", source, "error", err)
			return "", false
		}
		dir, err := f.Fetch(source, version)
		if err != nil {
			logger.Warn("module fetch failed; falling back to skip",
				"module", modName, "source", source, "error", err)
			return "", false
		}
		return dir, true
	}
	logger.Warn("module source isn't local and not in .terraform/modules/modules.json — "+
		"run `terraform init` so c3x can resolve it. Resources inside this module are missing from the estimate.",
		"module", modName, "source", source, "manifest_key", manifestKey)
	return "", false
}

func isLocalSource(s string) bool {
	return s == "." || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../")
}
