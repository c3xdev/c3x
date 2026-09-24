// Package terragrunt resolves a `terragrunt.hcl` into the directory
// and variable inputs the underlying Terraform module needs, then
// hands off to the Terraform parser. We deliberately don't shell out
// to the `terragrunt` binary — users would need it installed and on
// PATH at the right version, which is the kind of latent dependency
// that bites in CI.
//
// What we support today (the 80% of real configurations):
//
//   - terraform.source = "./local/path" → forwarded to terraform.ParseDirectory
//   - inputs = { name = value, ... }    → passed as Terraform variables
//   - locals { ... } resolved against the surrounding scope
//   - include "root" { path = find_in_parent_folders() } — partial:
//     we walk up looking for parent `terragrunt.hcl`s and merge their
//     inputs/locals
//
// What we don't yet:
//
//   - Remote sources (git/registry). Use `terragrunt init` first to
//     materialise them; we'll wire that path when go-getter integration
//     lands.
//   - `dependencies` blocks (cross-config dependency injection).
//   - `generate "..."` (writes auxiliary .tf files at apply time).
//   - `read_terragrunt_config()` — needs full Terragrunt semantics.
package terragrunt

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/terraform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// ConfigFile is the conventional Terragrunt filename. We auto-detect
// it in the input directory; users don't need to point at it
// explicitly.
const ConfigFile = "terragrunt.hcl"

// Options carry caller-supplied overrides parallel to those the
// Terraform parser accepts.
type Options struct {
	// Vars are CLI `--var name=value` overrides applied AFTER the
	// terragrunt-resolved inputs, so a user can override what
	// Terragrunt would have passed in.
	Vars   map[string]string
	Logger *slog.Logger
	// Offline is forwarded to the Terraform parser; disables network
	// module fetching inside the resolved module tree.
	Offline bool
	// AllowFileFunctions is forwarded to the Terraform parser.
	AllowFileFunctions bool
}

// ParseDirectory detects a Terragrunt config in `dir` and resolves
// it into the underlying Terraform module's resources. If the
// directory doesn't contain a terragrunt.hcl, returns an error so
// the dispatcher can fall back to plain Terraform parsing.
func ParseDirectory(dir string, opts Options) ([]domain.Resource, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	cfgPath := filepath.Join(dir, ConfigFile)
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, fmt.Errorf("no %s in %s", ConfigFile, dir)
	}
	resolved, err := resolveConfig(cfgPath, opts.Logger)
	if err != nil {
		return nil, err
	}
	if resolved.Source == "" {
		return nil, fmt.Errorf("%s: terraform.source is empty or not a local path", cfgPath)
	}
	moduleDir := resolveSourcePath(resolved.Source, dir)
	if info, err := os.Stat(moduleDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s: terraform.source %q does not resolve to a local directory (got %s)",
			cfgPath, resolved.Source, moduleDir)
	}

	// Compose the var map: terragrunt-resolved inputs first, then CLI
	// `--var` overrides (which always win).
	vars := make(map[string]string, len(resolved.Inputs)+len(opts.Vars))
	for k, v := range resolved.Inputs {
		vars[k] = stringifyCty(v)
	}
	for k, v := range opts.Vars {
		vars[k] = v
	}

	return terraform.ParseDirectory(moduleDir, terraform.Options{
		Vars:    vars,
		Logger:  opts.Logger,
		Offline: opts.Offline,

		AllowFileFunctions: opts.AllowFileFunctions,
	})
}

// IsTerragruntDirectory reports whether `dir` looks like a Terragrunt
// leaf (has a terragrunt.hcl). The dispatcher uses this to decide
// whether to invoke ParseDirectory or the Terraform parser directly.
func IsTerragruntDirectory(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ConfigFile))
	return err == nil
}

// resolvedConfig is what we extract from one (or merged) terragrunt.hcl
// files. Future fields (dependencies, generate, etc.) can extend the
// struct without changing the public Parse surface.
type resolvedConfig struct {
	Source string
	Inputs map[string]cty.Value
}

func resolveConfig(path string, logger *slog.Logger) (resolvedConfig, error) {
	cfg, err := parseFile(path)
	if err != nil {
		return resolvedConfig{}, err
	}

	// Walk includes upward, depth-first, merging parent values first
	// so child values override (matches Terragrunt's own semantics).
	merged := resolvedConfig{
		Inputs: map[string]cty.Value{},
	}
	for _, inc := range cfg.Includes {
		parentPath := resolveIncludePath(inc, path)
		if parentPath == "" {
			logger.Debug("terragrunt include skipped (unresolvable path)",
				"include", inc, "child", path)
			continue
		}
		parent, err := resolveConfig(parentPath, logger)
		if err != nil {
			logger.Warn("terragrunt parent include failed; continuing without it",
				"include", inc, "error", err)
			continue
		}
		if parent.Source != "" {
			merged.Source = parent.Source
		}
		for k, v := range parent.Inputs {
			merged.Inputs[k] = v
		}
	}

	// Now layer this file's own values on top.
	if cfg.Source != "" {
		merged.Source = cfg.Source
	}
	for k, v := range cfg.Inputs {
		merged.Inputs[k] = v
	}
	return merged, nil
}

// parsedConfig is the raw shape of one terragrunt.hcl after HCL
// decoding. Conversion into a `resolvedConfig` happens in
// resolveConfig where merge precedence lives.
type parsedConfig struct {
	Source   string
	Inputs   map[string]cty.Value
	Includes []includeRef
}

type includeRef struct {
	// Path is the literal value `path = "..."` if it was a string;
	// FindInParentFolders is true when the path used the helper.
	Path                string
	FindInParentFolders bool
}

func parseFile(path string) (parsedConfig, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return parsedConfig{}, fmt.Errorf("parsing %s: %s", path, diags.Error())
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return parsedConfig{}, fmt.Errorf("%s: unexpected HCL body type %T", path, file.Body)
	}

	cfg := parsedConfig{
		Inputs: map[string]cty.Value{},
	}
	ctx := evalContext()

	// First pass: resolve locals so the second pass can reference them.
	locals := resolveLocals(body, ctx)
	if len(locals) > 0 {
		ctx.Variables["local"] = cty.ObjectVal(locals)
	}

	// Top-level attributes.
	for _, attr := range body.Attributes {
		if attr.Name != "inputs" {
			continue
		}
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return parsedConfig{}, fmt.Errorf("%s: inputs: %s", path, diags.Error())
		}
		if val.Type().IsObjectType() || val.Type().IsMapType() {
			it := val.ElementIterator()
			for it.Next() {
				k, v := it.Element()
				cfg.Inputs[k.AsString()] = v
			}
		}
	}

	// Blocks: `terraform { ... }`, `include "..." { ... }`, etc.
	for _, block := range body.Blocks {
		switch block.Type {
		case "terraform":
			if src, ok := block.Body.Attributes["source"]; ok {
				val, diags := src.Expr.Value(ctx)
				if diags.HasErrors() {
					return parsedConfig{}, fmt.Errorf("%s: terraform.source: %s", path, diags.Error())
				}
				if val.Type() == cty.String && !val.IsNull() {
					cfg.Source = val.AsString()
				}
			}
		case "include":
			cfg.Includes = append(cfg.Includes, decodeInclude(block))
		}
	}
	return cfg, nil
}

func decodeInclude(block *hclsyntax.Block) includeRef {
	ref := includeRef{}
	pathAttr, ok := block.Body.Attributes["path"]
	if !ok {
		return ref
	}
	// `path` is often `find_in_parent_folders()` which we can't
	// evaluate in HCL (it's a Terragrunt-specific function). Detect
	// the literal call shape via the raw source range; if it doesn't
	// match, try evaluating as a normal string expression.
	src := strings.TrimSpace(rangeText(pathAttr.Expr.Range()))
	if strings.HasPrefix(src, "find_in_parent_folders(") {
		ref.FindInParentFolders = true
		return ref
	}
	val, diags := pathAttr.Expr.Value(evalContext())
	if diags.HasErrors() {
		return ref
	}
	if val.Type() == cty.String && !val.IsNull() {
		ref.Path = val.AsString()
	}
	return ref
}

// resolveIncludePath turns an `include` block into an absolute path
// to the parent terragrunt.hcl. find_in_parent_folders() walks upward
// from the child's directory looking for terragrunt.hcl; literal
// paths resolve relative to the child.
func resolveIncludePath(inc includeRef, childPath string) string {
	if inc.FindInParentFolders {
		dir := filepath.Dir(childPath)
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				return ""
			}
			candidate := filepath.Join(parent, ConfigFile)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			dir = parent
		}
	}
	if inc.Path == "" {
		return ""
	}
	if filepath.IsAbs(inc.Path) {
		return inc.Path
	}
	return filepath.Join(filepath.Dir(childPath), inc.Path)
}

// resolveLocals evaluates `locals { ... }` blocks via fixed-point
// iteration so locals can reference earlier-defined locals in any
// declaration order. Mirrors the Terraform parser's behaviour.
func resolveLocals(body *hclsyntax.Body, ctx *hcl.EvalContext) map[string]cty.Value {
	type pending struct {
		name string
		expr hclsyntax.Expression
	}
	var queue []pending
	for _, block := range body.Blocks {
		if block.Type != "locals" {
			continue
		}
		for _, attr := range block.Body.Attributes {
			queue = append(queue, pending{name: attr.Name, expr: attr.Expr})
		}
	}
	out := map[string]cty.Value{}
	for {
		progress := false
		var still []pending
		for _, p := range queue {
			// Re-inject the growing locals into the evaluator before
			// each attempt so a later prefix can read the earlier env.
			ctx.Variables["local"] = cty.ObjectVal(merge(out))
			val, diags := p.expr.Value(ctx)
			if diags.HasErrors() {
				still = append(still, p)
				continue
			}
			out[p.name] = val
			progress = true
		}
		queue = still
		if !progress || len(queue) == 0 {
			break
		}
	}
	return out
}

// merge clones the locals map into the cty.Object shape the evaluator
// expects. Building a fresh object each pass means evaluator caches
// don't pin a stale "local has nothing yet" view.
func merge(m map[string]cty.Value) map[string]cty.Value {
	if len(m) == 0 {
		return map[string]cty.Value{}
	}
	out := make(map[string]cty.Value, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// resolveSourcePath turns the `terraform.source` value into the
// directory we should hand to the Terraform parser. Local-relative
// sources resolve relative to the terragrunt.hcl's directory; the
// `//` separator that Terragrunt uses for nested module paths is
// honoured (we trim it to a plain path because we descend the whole
// tree anyway).
func resolveSourcePath(source, baseDir string) string {
	source = strings.TrimSpace(source)
	// Terragrunt allows `path//submodule` to point to a sub-tree;
	// for our purposes the submodule is the module we want.
	if i := strings.Index(source, "//"); i >= 0 {
		// `git::https://...//modules/vpc` → can't resolve locally.
		// `./modules//foo` → we want `./modules/foo`.
		// Distinguish on whether the leading segment is a URL.
		head := source[:i]
		if strings.Contains(head, "://") || strings.HasPrefix(head, "git@") {
			// Remote source. Return verbatim so the caller's stat()
			// fails with a clear "not a directory" message.
			return source
		}
		source = head + "/" + source[i+2:]
	}
	if filepath.IsAbs(source) {
		return source
	}
	return filepath.Join(baseDir, source)
}

// rangeText extracts the source text of an HCL expression. Used to
// recognise the `find_in_parent_folders()` call shape without
// evaluating it (HCL would error because the function isn't
// registered in our eval context).
func rangeText(r hcl.Range) string {
	raw, err := os.ReadFile(r.Filename)
	if err != nil {
		return ""
	}
	if r.Start.Byte < 0 || r.End.Byte > len(raw) || r.End.Byte < r.Start.Byte {
		return ""
	}
	return string(raw[r.Start.Byte:r.End.Byte])
}

// evalContext returns the HCL evaluator with no Terragrunt-specific
// functions registered yet. Adding `find_in_parent_folders()`,
// `path_relative_to_include()`, etc. is straightforward — register
// them as `function.Function` values — is future scope.
func evalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{},
	}
}

// stringifyCty renders a cty.Value as the string a Terraform `--var`
// would carry. We use the GoString form for numbers/bools because
// the Terraform parser's parseCLIVar re-parses these as HCL.
func stringifyCty(v cty.Value) string {
	if v.IsNull() || !v.IsKnown() {
		return ""
	}
	switch v.Type() {
	case cty.String:
		return fmt.Sprintf("%q", v.AsString())
	case cty.Bool:
		if v.True() {
			return "true"
		}
		return "false"
	case cty.Number:
		return v.AsBigFloat().Text('g', -1)
	}
	// Lists / maps / objects: render with goCty so the Terraform
	// parser receives a re-parseable HCL literal.
	return v.GoString()
}
