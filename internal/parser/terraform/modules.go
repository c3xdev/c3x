package terraform

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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

// expandModules walks every `module "X" { ... }` block and recursively
// parses each module's source directory, threading parent inputs in as
// the child's variables and prefixing emitted Resources with the
// `module.X.` path.
//
// Local sources resolve via the filesystem; registry/git sources use
// the manifest produced by `terraform init`. Sources we can't resolve
// (no manifest entry) log a loud warning and are skipped — the rest of
// the config still produces a useful estimate.
//
//nolint:gocyclo // The pipeline is naturally a few flat branches.
func expandModules(
	baseDir string,
	sources []sourceFile,
	parentVars map[string]cty.Value,
	parentLocals map[string]cty.Value,
	parentData cty.Value,
	parentRegion string,
	namePrefix string,
	keyPrefix string,
	initModules map[string]string,
	depth int,
	offline bool,
	logger *slog.Logger,
	out *[]domain.Resource,
) error {
	if depth >= MaxModuleDepth {
		logger.Warn("module nesting exceeded MaxModuleDepth; truncating expansion. "+
			"likely a self-referential `module \"x\" { source = \".\" }`",
			"depth", depth, "max", MaxModuleDepth)
		return nil
	}
	parentCtx := buildEvalContext(asObject(parentVars), asObject(parentLocals), parentData, nil)

	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "module" || len(block.Labels) == 0 {
				continue
			}
			modName := block.Labels[0]

			source, ok := readModuleSource(block.Body, parentCtx)
			if !ok {
				continue
			}
			version := readModuleVersion(block.Body, parentCtx)

			manifestKey := modName
			if keyPrefix != "" {
				manifestKey = keyPrefix + "." + modName
			}

			childDir, ok := resolveModuleSource(source, version, baseDir, manifestKey, initModules, modName, offline, logger)
			if !ok {
				continue
			}
			if info, err := os.Stat(childDir); err != nil || !info.IsDir() {
				logger.Warn("module source path is not a directory; skipping",
					"module", modName, "dir", childDir)
				continue
			}

			// Evaluate every non-meta attribute against the parent's
			// scope to build the child's var.x inputs.
			childInputs := map[string]cty.Value{}
			for _, attr := range block.Body.Attributes {
				switch attr.Name {
				case "source", "version", "count", "for_each", "providers", "depends_on":
					continue
				}
				val, diags := attr.Expr.Value(parentCtx)
				if diags.HasErrors() {
					logger.Debug("module input evaluation failed",
						"module", modName, "input", attr.Name,
						"diags", formatDiags(diags))
					continue
				}
				childInputs[attr.Name] = val
			}

			childPaths, err := filepath.Glob(filepath.Join(childDir, "*.tf"))
			if err != nil {
				return fmt.Errorf("module %s glob: %w", modName, err)
			}
			sort.Strings(childPaths)
			childSources, err := loadFiles(childPaths)
			if err != nil {
				return fmt.Errorf("module %s load: %w", modName, err)
			}

			childVars, err := collectVariableDefaults(childSources)
			if err != nil {
				return err
			}
			if err := applyAutoTfvars(childDir, childVars); err != nil {
				return err
			}
			for k, v := range childInputs {
				childVars[k] = v
			}

			// Fill in optional() defaults from the variable type
			// constraints. This bridges the gap between Terraform's
			// runtime type-system and c3x's static parsing: attributes
			// like `os_disk = optional(object({disk_size_gb = optional(number, 64)}), {})`
			// get their defaults applied to the caller-supplied values.
			applyOptionalDefaults(childSources, childVars)

			childData := collectDataBlocks(childSources)
			childLocals := resolveLocals(childSources, childVars, childData)
			childRegion := findDefaultRegion(childSources, childVars, childLocals, childData, logger)
			if childRegion == "" {
				childRegion = parentRegion
			}

			childPrefix := namePrefix + "module." + modName + "."
			for _, csrc := range childSources {
				for _, cb := range csrc.Body.Blocks {
					if cb.Type != "resource" || len(cb.Labels) < 2 {
						continue
					}
					kind := cb.Labels[0]
					nm := cb.Labels[1]
					if err := emitOne(
						csrc.Path, kind, nm, cb.Body,
						childVars, childLocals, childData, childRegion,
						childPrefix, logger, out,
					); err != nil {
						return err
					}
				}
			}

			if err := expandModules(
				childDir, childSources,
				childVars, childLocals, childData, childRegion,
				childPrefix, manifestKey, initModules, depth+1,
				offline, logger, out,
			); err != nil {
				return err
			}
		}
	}
	return nil
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
	if diags.HasErrors() || val.Type() != cty.String || val.IsNull() {
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
