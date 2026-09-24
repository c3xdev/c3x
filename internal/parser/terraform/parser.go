// Package terraform parses Terraform and OpenTofu configurations (.tf,
// .tofu, their .json forms, and .hcl) into domain.Resources. The pipeline
// is deliberately staged so each phase's inputs are explicit:
//
//	files          configuration files on disk, override files merged in
//	  → variables  variable defaults + tfvars + CLI overrides
//	  → data       placeholder shapes for data blocks
//	  → locals     fixed-point resolution against (var, data)
//	  → resources  count/for_each expansion + attribute evaluation
//	  → modules    recursive parse of every module block, per instance
//
// All HCL evaluation goes through buildEvalContext so the function
// library and scope layout stay consistent across stages.
package terraform

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Options are caller-supplied overrides layered on top of the on-disk
// configuration.
type Options struct {
	VarFiles []string
	Vars     map[string]string
	Logger   *slog.Logger
	// Offline disables network module fetching (Terraform Registry,
	// Git, HTTP archives). Local module paths and pre-resolved
	// .terraform/modules entries still resolve. Set by `--offline`
	// estimates; tests should always set it so module resolution
	// never surprises CI with a network call.
	Offline bool
	// AllowFileFunctions registers file(), templatefile(), fileset() and
	// the other filesystem functions, confined to the project directory.
	// Off by default; see config.Resolved.AllowFileFunctions for why.
	AllowFileFunctions bool
	// Untrusted marks input the runner does not control (untrusted-input
	// mode): local module sources are confined to the root directory.
	Untrusted bool
}

// ParseDirectory loads every `*.tf` / `*.tf.json` and OpenTofu `*.tofu` /
// `*.tofu.json` file in `dir` (see configFiles), merges override files
// into the blocks they override, and runs the full parse pipeline against
// the combined config.
func ParseDirectory(dir string, opts Options) ([]domain.Resource, error) {
	sources, err := loadConfigDir(dir, loggerOf(opts))
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no .tf, .tf.json, .tofu or .tofu.json files found in %s", dir)
	}
	return parseSources(sources, dir, opts)
}

// ParseFile parses a single `.tf`, `.tofu`, `.hcl`, `.tf.json` or
// `.tofu.json` file. Convenience for tests and single-file invocations.
func ParseFile(path string, opts Options) ([]domain.Resource, error) {
	sources, err := loadFiles([]string{path}, loggerOf(opts))
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	return parseSources(sources, dir, opts)
}

func loggerOf(opts Options) *slog.Logger {
	if opts.Logger == nil {
		return slog.Default()
	}
	return opts.Logger
}

// sourceFile pairs a parsed file with the path it came from so warnings
// can cite the actual source location.
type sourceFile struct {
	Path string
	File *hcl.File
	Body *hclsyntax.Body
}

// loadConfigDir loads a module directory the way Terraform does: every
// configuration file, then each override file merged into the blocks it
// overrides.
func loadConfigDir(dir string, logger *slog.Logger) ([]sourceFile, error) {
	paths, err := configFiles(dir)
	if err != nil {
		return nil, err
	}
	primaryPaths, overridePaths := splitOverrides(paths)
	primary, err := loadFiles(primaryPaths, logger)
	if err != nil {
		return nil, err
	}
	overrides, err := loadFiles(overridePaths, logger)
	if err != nil {
		return nil, err
	}
	mergeOverrides(primary, overrides, logger)
	return primary, nil
}

// loadFiles parses each file. Native syntax is parsed directly; JSON
// syntax (.tf.json / .tofu.json) is first translated to native syntax
// (see jsonToNative), so every later stage works on one AST type.
func loadFiles(paths []string, logger *slog.Logger) ([]sourceFile, error) {
	parser := hclparse.NewParser()
	out := make([]sourceFile, 0, len(paths))
	for _, p := range paths {
		var f *hcl.File
		var diags hcl.Diagnostics
		if isJSONConfig(p) {
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", p, err)
			}
			native, err := jsonToNative(raw, p, logger)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", p, err)
			}
			f, diags = parser.ParseHCL(native, p)
			if diags.HasErrors() {
				// The translation produced something HCL rejects: a gap
				// in jsonToNative, not the user's error. Say so, and
				// price the rest.
				logger.Warn("c3x could not translate this JSON configuration file; "+
					"resources declared in it are missing from the estimate",
					"file", p, "error", formatDiags(diags))
				continue
			}
		} else {
			f, diags = parser.ParseHCLFile(p)
			if diags.HasErrors() {
				return nil, fmt.Errorf("parsing %s: %s", p, formatDiags(diags))
			}
		}
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			return nil, fmt.Errorf("%s: unexpected HCL body type %T", p, f.Body)
		}
		out = append(out, sourceFile{Path: p, File: f, Body: body})
	}
	return out, nil
}

// parseSources is the pipeline driver. Stages are explicit so each one
// can be reasoned about in isolation; the variables/locals/data/region
// state objects are owned here and passed down.
func parseSources(sources []sourceFile, baseDir string, opts Options) ([]domain.Resource, error) {
	logger := loggerOf(opts)

	// Where expressions are evaluated from: path.module / path.root and
	// the filesystem functions, confined to the project directory.
	scope := newRootScope(baseDir, opts.AllowFileFunctions, opts.Untrusted)

	// Stage 0: terraform-init module manifest, if present.
	initModules := loadInitModules(baseDir, logger)

	// Stage 1: variable defaults across every file.
	variables, err := collectVariableDefaults(sources)
	if err != nil {
		return nil, err
	}

	// Stage 2: tfvars layers (auto + explicit + CLI).
	if err := applyAutoTfvars(baseDir, variables); err != nil {
		return nil, fmt.Errorf("auto-tfvars: %w", err)
	}
	for _, vf := range opts.VarFiles {
		if err := applyVarFile(vf, variables); err != nil {
			return nil, fmt.Errorf("--var-file %s: %w", vf, err)
		}
	}
	for k, v := range opts.Vars {
		variables[k] = parseCLIVar(v)
	}

	// Stage 2b: normalise each variable to its declared `type` constraint,
	// filling optional() object-attribute defaults the way Terraform does
	// before anything reads the value. This covers both `optional(t, def)`
	// (explicit default) and bare `optional(t)` (materialised as null), so
	// downstream traversals like `each.value.x` resolve instead of erroring
	// and being swallowed by try().
	applyVariableTypes(variables, collectVariableTypes(sources))

	// Stage 3: data-block placeholders for `data.kind.name.attr`
	// traversals. Placeholders that depend on a region (availability
	// zones) take it from provider blocks resolvable from variables alone,
	// since locals aren't resolved yet.
	regionHints := collectProviderRegions(sources, variables, nil, cty.EmptyObjectVal, "", logger)
	data := collectDataBlocks(sources, regionHints)

	// Stage 4: locals — fixed-point against (var, data).
	locals := resolveLocals(scope, sources, variables, data, logger)

	// Stage 5: provider regions (after vars/locals/data are populated so
	// `provider "aws" { region = var.region }` resolves). Each resource is
	// priced in its own provider's region; findDefaultRegion is the
	// fallback for resources whose provider can't be resolved.
	regions := collectProviderRegions(sources, variables, locals, data,
		findDefaultRegion(sources, variables, locals, data, logger), logger)

	env := moduleEnv{
		scope:     scope,
		vars:      variables,
		locals:    locals,
		data:      data,
		resources: collectLiteralResources(sources),
		regions:   regions,
		logger:    logger,
	}

	// Stage 6: walk resource blocks; each contributes one or more
	// domain.Resource entries depending on count / for_each.
	var resources []domain.Resource
	if err := emitResources(env, sources, "", &resources); err != nil {
		return nil, err
	}

	// Stage 7: recursively expand modules.
	expander := moduleExpander{initModules: initModules, offline: opts.Offline, logger: logger, out: &resources}
	if err := expander.expandModules(env, baseDir, sources, "", "", 0); err != nil {
		return nil, err
	}

	return resources, nil
}

// errInvalidVarFile is returned when a tfvars file is malformed.
type errInvalidVarFile struct {
	path string
	err  error
}

func (e *errInvalidVarFile) Error() string { return fmt.Sprintf("%s: %v", e.path, e.err) }
func (e *errInvalidVarFile) Unwrap() error { return e.err }

// fileExists is the convenience guard used by stage helpers that want
// to silently skip optional files (auto-tfvars, modules.json) rather
// than fail when they're absent.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
