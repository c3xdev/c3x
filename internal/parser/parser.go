// Package parser turns IaC sources into domain.Resources. The
// dispatcher in this file picks the right backend by inspecting the
// path; per-format work lives in the subpackages:
//
//	parser/terraform        — .tf / .tofu / .hcl files (and directories of them)
//	parser/plan             — terraform-show -json output
//	parser/cloudformation   — CloudFormation YAML / JSON
//
// Every backend produces []domain.Resource, so the calculator never
// learns which format the user supplied.
package parser

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/cloudformation"
	"github.com/c3xdev/c3x/internal/parser/plan"
	"github.com/c3xdev/c3x/internal/parser/terraform"
	"github.com/c3xdev/c3x/internal/parser/terragrunt"
)

// Options are caller-supplied inputs that augment the on-disk
// configuration. Plan-JSON input ignores most of these because plan
// files already have resolved values baked in.
type Options struct {
	// VarFiles are explicit `-var-file` paths, applied after the
	// directory's auto-tfvars files. Terraform-only.
	VarFiles []string
	// Vars are CLI `--var name=value` overrides, applied last.
	// Re-used as CloudFormation Parameter overrides — the syntax is
	// language-agnostic.
	Vars map[string]string
	// Region is the deployment region. Required for CloudFormation
	// (templates don't carry one); falls back to a per-cloud
	// provider-block value for Terraform.
	Region string
	// Logger receives parse-time diagnostics. Defaults to slog.Default().
	Logger *slog.Logger
	// Offline disables network module fetching in the Terraform
	// backend. Mapped from the CLI's --offline flag.
	Offline bool
	// AllowFileFunctions registers file(), templatefile(), fileset() and
	// the other filesystem functions, confined to the project directory.
	// Off by default; see config.Resolved.AllowFileFunctions for why.
	AllowFileFunctions bool
	// Untrusted marks input the runner does not control; see
	// terraform.Options.Untrusted.
	Untrusted bool
}

// Parse auto-detects the input type and returns the parsed resources.
//
// Detection rules:
//   - directory                    → Terraform .tf / OpenTofu .tofu files
//   - .tf / .tofu / .hcl           → single Terraform or OpenTofu file
//   - .json                        → Terraform plan JSON (Terraform shape
//     detected by `resource_changes` key
//     pre-emptively; CloudFormation JSON
//     must use .cfn / .cfn.json to
//     disambiguate, or be passed via
//     --format cfn)
//   - .yaml / .yml                 → CloudFormation
//   - .cfn / .cfn.yaml / .cfn.json → CloudFormation (explicit)
//
// Anything else is rejected explicitly so users don't get a generic
// "no resources found" when they pointed at the wrong path.
func Parse(path string, opts Options) ([]domain.Resource, error) {
	if path == "" {
		return nil, errors.New("parser.Parse: path is empty")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	resources, err := parseRaw(path, opts)
	if err != nil {
		return nil, err
	}
	// Copy attributes that live on a related resource onto the resource
	// billed for them (see inherit.go). Done here so every command that
	// parses goes through it, and so the catalog can stay on expressions
	// that every released client understands.
	applyInheritance(resources)
	return resources, nil
}

// parseRaw is Parse without the cross-resource enrichment.
func parseRaw(path string, opts Options) ([]domain.Resource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		if terragrunt.IsTerragruntDirectory(path) {
			return terragrunt.ParseDirectory(path, terragrunt.Options{
				Vars:    opts.Vars,
				Logger:  opts.Logger,
				Offline: opts.Offline,

				AllowFileFunctions: opts.AllowFileFunctions,
				Untrusted:          opts.Untrusted,
			})
		}
		return terraform.ParseDirectory(path, toTerraformOptions(opts))
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tf"), strings.HasSuffix(lower, ".tofu"), strings.HasSuffix(lower, ".hcl"):
		return terraform.ParseFile(path, toTerraformOptions(opts))
	case strings.HasSuffix(lower, ".cfn"),
		strings.HasSuffix(lower, ".cfn.yaml"),
		strings.HasSuffix(lower, ".cfn.yml"),
		strings.HasSuffix(lower, ".cfn.json"),
		strings.HasSuffix(lower, ".yaml"),
		strings.HasSuffix(lower, ".yml"):
		return cloudformation.ParseFile(path, toCFNOptions(opts))
	case strings.HasSuffix(lower, ".json"):
		// Disambiguate: Terraform plan JSON contains "resource_changes";
		// CloudFormation JSON contains "Resources". Sniff the file
		// instead of forcing the user to remember a flag.
		if isCFNJSON(path) {
			return cloudformation.ParseFile(path, toCFNOptions(opts))
		}
		return plan.ParseFile(path, opts.Logger)
	default:
		return nil, fmt.Errorf("unsupported input %q (want a directory, .tf, .tofu, .hcl, .yaml, .yml, .json)",
			filepath.Base(path))
	}
}

// isPlanFile reports whether path is a Terraform plan JSON (a .json file
// that isn't a CloudFormation template). Shared by [PlanBaseline] and
// [ParsePostApply] so plan detection stays in one place.
// IsPlanFile reports whether path is a Terraform or OpenTofu plan JSON
// (a .json file that is not a CloudFormation template).
func IsPlanFile(path string) bool { return isPlanFile(path) }

func isPlanFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".json") && !isCFNJSON(path)
}

// PlanBaseline returns the pre-apply ("before") resource set when path is
// a Terraform plan JSON that carries prior state, so callers can render a
// cost delta straight from the plan with no --baseline file. For any
// other input (a directory, .tf, CloudFormation), or a greenfield plan
// with no before-state, ok is false and the caller should fall back to
// the absolute estimate. The plan is read twice (once here, once in
// [Parse]); plan JSON is small and this keeps the two call sites simple.
func PlanBaseline(path string, opts Options) ([]domain.Resource, bool, error) {
	if path == "" || !isPlanFile(path) {
		return nil, false, nil
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	out, hasBaseline, err := plan.ParseBaselineFile(path, opts.Logger)
	if err != nil {
		return nil, false, err
	}
	applyInheritance(out)
	return out, hasBaseline, nil
}

// ParsePostApply returns the strictly post-apply resource set. For a
// Terraform plan JSON it excludes resources scheduled for deletion (they
// won't exist after apply), unlike [Parse], which appends them for the
// single-estimate --show-delta view. For every other input it is
// identical to [Parse]. Used as the "current" side of the plan-aware diff
// so removed resources land only in the before set and the absolute total
// doesn't count them.
func ParsePostApply(path string, opts Options) ([]domain.Resource, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if isPlanFile(path) {
		out, err := plan.ParsePostApplyFile(path, opts.Logger)
		if err != nil {
			return nil, err
		}
		applyInheritance(out)
		return out, nil
	}
	return Parse(path, opts)
}

func toTerraformOptions(o Options) terraform.Options {
	return terraform.Options{
		VarFiles: o.VarFiles,
		Vars:     o.Vars,
		Logger:   o.Logger,
		Offline:  o.Offline,

		AllowFileFunctions: o.AllowFileFunctions,
		Untrusted:          o.Untrusted,
	}
}

func toCFNOptions(o Options) cloudformation.Options {
	// Re-purpose --var as CloudFormation Parameter overrides because
	// the syntax (NAME=VALUE) is identical and users shouldn't have to
	// learn a new flag for the same intent.
	params := make(map[string]any, len(o.Vars))
	for k, v := range o.Vars {
		params[k] = v
	}
	return cloudformation.Options{
		Parameters: params,
		Region:     o.Region,
		Logger:     o.Logger,
	}
}

// isCFNJSON peeks at a JSON file to decide whether it looks like a
// CloudFormation template (has a top-level "Resources" key) versus a
// Terraform plan (has "resource_changes"). The check is cheap — we
// only need the first ~1 KB.
func isCFNJSON(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Use whichever signature appears first — plan JSON puts
	// `format_version` before `resource_changes`, CFN puts
	// `AWSTemplateFormatVersion` before `Resources`. Both are
	// distinctive enough to use as a sniff.
	for i := 0; i < len(raw) && i < 2048; i++ {
		if raw[i] == '"' {
			rest := string(raw[i:])
			if strings.HasPrefix(rest, `"AWSTemplateFormatVersion"`) ||
				strings.HasPrefix(rest, `"Resources"`) {
				return true
			}
			if strings.HasPrefix(rest, `"resource_changes"`) ||
				strings.HasPrefix(rest, `"format_version"`) {
				return false
			}
		}
	}
	return false
}
