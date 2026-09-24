// Package terraform parses Terraform and OpenTofu .tf / .tofu / .hcl configurations into
// domain.Resources. The pipeline is deliberately staged so each phase's
// inputs are explicit:
//
//	files          .tf files on disk
//	  → variables  variable defaults + tfvars + CLI overrides
//	  → locals     fixed-point resolution against (var, data)
//	  → data       placeholder shapes for data blocks
//	  → resources  count/for_each expansion + attribute evaluation
//	  → modules    recursive parse of every module block
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
}

// ParseDirectory loads every `*.tf` and OpenTofu `*.tofu` file in `dir`
// (see configFiles) and runs the full parse pipeline against the combined
// config.
func ParseDirectory(dir string, opts Options) ([]domain.Resource, error) {
	matches, err := configFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .tf or .tofu files found in %s", dir)
	}
	sources, err := loadFiles(matches)
	if err != nil {
		return nil, err
	}
	return parseSources(sources, dir, opts)
}

// ParseFile parses a single `.tf`, `.tofu` or `.hcl` file. Convenience for tests
// and single-file invocations.
func ParseFile(path string, opts Options) ([]domain.Resource, error) {
	sources, err := loadFiles([]string{path})
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	return parseSources(sources, dir, opts)
}

// sourceFile pairs a parsed file with the path it came from so warnings
// can cite the actual source location.
type sourceFile struct {
	Path string
	File *hcl.File
	Body *hclsyntax.Body
}

func loadFiles(paths []string) ([]sourceFile, error) {
	parser := hclparse.NewParser()
	out := make([]sourceFile, 0, len(paths))
	for _, p := range paths {
		f, diags := parser.ParseHCLFile(p)
		if diags.HasErrors() {
			return nil, fmt.Errorf("parsing %s: %s", p, formatDiags(diags))
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
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

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

	// Stage 3: data-block placeholders for `data.kind.name.attr` traversals.
	data := collectDataBlocks(sources)

	// Stage 4: locals — fixed-point against (var, data).
	locals := resolveLocals(sources, variables, data, logger)

	// Stage 5: provider regions (after vars/locals/data are populated so
	// `provider "aws" { region = var.region }` resolves). Each resource is
	// priced in its own provider's region; findDefaultRegion is the
	// fallback for resources whose provider can't be resolved.
	regions := collectProviderRegions(sources, variables, locals, data,
		findDefaultRegion(sources, variables, locals, data, logger), logger)

	// Stage 6: walk resource blocks; each contributes one or more
	// domain.Resource entries depending on count / for_each.
	resources, err := emitResources(sources, variables, locals, data, regions, logger)
	if err != nil {
		return nil, err
	}

	// Stage 7: recursively expand modules.
	if err := expandModules(
		baseDir, sources,
		variables, locals, data, regions,
		"", "", initModules, 0,
		opts.Offline, logger, &resources,
	); err != nil {
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
