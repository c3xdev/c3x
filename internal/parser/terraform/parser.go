// Package terraform parses Terraform .tf / .hcl configurations into
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
	"sort"

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

// ParseDirectory loads every `*.tf` file in `dir` and runs the full
// parse pipeline against the combined config.
func ParseDirectory(dir string, opts Options) ([]domain.Resource, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .tf files found in %s", dir)
	}
	sort.Strings(matches)
	sources, err := loadFiles(matches)
	if err != nil {
		return nil, err
	}
	return parseSources(sources, dir, opts)
}

// ParseFile parses a single `.tf` or `.hcl` file. Convenience for tests
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

	// Stage 2b: apply optional() defaults from variable type constraints.
	// This fills in attributes like `optional(string, "default")` for
	// root-module variables, matching Terraform's runtime behavior.
	applyOptionalDefaults(sources, variables)

	// Stage 3: data-block placeholders for `data.kind.name.attr` traversals.
	data := collectDataBlocks(sources)

	// Stage 4: locals — fixed-point against (var, data).
	locals := resolveLocals(sources, variables, data)

	// Stage 5: default region (after vars/locals/data are populated so
	// `provider "aws" { region = var.region }` resolves).
	region := findDefaultRegion(sources, variables, locals, data, logger)

	// Stage 6: walk resource blocks; each contributes one or more
	// domain.Resource entries depending on count / for_each.
	resources, err := emitResources(sources, variables, locals, data, region, logger)
	if err != nil {
		return nil, err
	}

	// Stage 7: recursively expand modules.
	if err := expandModules(
		baseDir, sources,
		variables, locals, data, region,
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
