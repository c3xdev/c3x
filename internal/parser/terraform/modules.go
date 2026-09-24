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

// moduleExpander carries what stays fixed while expanding a module tree.
type moduleExpander struct {
	initModules map[string]string
	offline     bool
	logger      *slog.Logger
	out         *[]domain.Resource
}

// expandModules walks every `module "X" { ... }` block and recursively
// parses each module's source directory, threading parent inputs in as
// the child's variables and prefixing emitted Resources with the
// `module.X.` path.
//
// A module block with count or for_each is expanded once per instance,
// with count.index / each.key / each.value in scope for its inputs, and
// its resources are addressed module.X[0]. / module.X["a"]., as Terraform
// addresses them. Every instance counts against the parse budget.
//
// Local sources resolve via the filesystem; registry/git sources use
// the manifest produced by `terraform init`. Sources we can't resolve
// (no manifest entry) log a loud warning and are skipped — the rest of
// the config still produces a useful estimate.
func (m moduleExpander) expandModules(
	env moduleEnv,
	baseDir string,
	sources []sourceFile,
	namePrefix string,
	keyPrefix string,
	depth int,
) error {
	logger := m.logger
	if depth >= MaxModuleDepth {
		logger.Warn("module nesting exceeded MaxModuleDepth; truncating expansion. "+
			"likely a self-referential `module \"x\" { source = \".\" }`",
			"depth", depth, "max", MaxModuleDepth)
		return nil
	}
	parentCtx := env.evalContext(nil)

	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "module" || len(block.Labels) == 0 {
				continue
			}
			modName := block.Labels[0]
			address := namePrefix + "module." + modName

			instances, ok := expandInstances(env, block.Body, src.Path, address)
			if !ok {
				continue
			}
			if err := env.scope.budget.moduleCall(address, len(instances)); err != nil {
				return err
			}
			if len(instances) == 0 {
				continue
			}

			source, ok := readModuleSource(block.Body, parentCtx)
			if !ok {
				continue
			}
			version := readModuleVersion(block.Body, parentCtx)

			// The init manifest keys modules by name, never by instance.
			manifestKey := modName
			if keyPrefix != "" {
				manifestKey = keyPrefix + "." + modName
			}

			childDir, ok := resolveModuleSource(source, version, baseDir, manifestKey, m.initModules, modName, m.offline, logger)
			if !ok {
				continue
			}
			if info, err := os.Stat(childDir); err != nil || !info.IsDir() {
				logger.Warn("module source path is not a directory; skipping",
					"module", modName, "dir", childDir)
				continue
			}
			// Untrusted input may not pull configuration from outside its own
			// tree: on a server, source = "../other-upload" (or a committed
			// .terraform/modules/modules.json pointing there) would fold
			// another user's infrastructure into this estimate. Terraform
			// allows ../ sources, so trusted parses keep doing the same.
			if env.scope.untrusted && !env.scope.containsDir(childDir) {
				logger.Warn("untrusted input: module source is outside the project directory; skipping",
					"module", modName, "dir", childDir)
				continue
			}

			childSources, err := loadConfigDir(childDir, logger)
			if err != nil {
				return fmt.Errorf("module %s: %w", modName, err)
			}
			child := moduleSource{
				dir:       childDir,
				sources:   childSources,
				types:     collectVariableTypes(childSources),
				resources: collectLiteralResources(childSources),
			}
			for _, inst := range instances {
				if err := m.expandInstance(
					env, block, child, inst,
					address+inst.suffix+".", manifestKey, depth,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// moduleSource is a module's configuration, loaded once per module block
// and shared by every instance of it.
type moduleSource struct {
	dir       string
	sources   []sourceFile
	types     map[string]variableTypeConstraint
	resources map[string]cty.Value
}

// expandInstance parses one instance of a module call: its inputs are
// evaluated in the parent's scope plus the instance's count / each,
// then the child's resources are emitted under childPrefix and its own
// module calls recursed into.
func (m moduleExpander) expandInstance(
	parent moduleEnv,
	block *hclsyntax.Block,
	child moduleSource,
	inst instance,
	childPrefix, manifestKey string,
	depth int,
) error {
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
		return err
	}
	if err := applyAutoTfvars(child.dir, childVars); err != nil {
		return err
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
	childLocals := resolveLocals(childScope, child.sources, childVars, childData, logger)
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
	}
	if err := emitResources(env, child.sources, childPrefix, m.out); err != nil {
		return err
	}
	return m.expandModules(env, child.dir, child.sources, childPrefix, manifestKey, depth+1)
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
