// Package usage loads a `c3x-usage.yml` file and applies its
// runtime-usage quantities onto parsed resources. Catalog files
// describe HOW a resource bills; the usage file describes WHAT the
// user actually uses (`monthly_requests = 1_500_000`,
// `standard_storage_gb = 500`).
//
// The file is intentionally separate from `.tfvars` because runtime
// usage is observability data — it doesn't change the infrastructure,
// it changes the cost model's inputs. Mixing the two into Terraform
// variables conflates infra-state with observability.
//
// Schema:
//
//	version: 0.1
//	resource_usage:
//	  aws_lambda_function.api:
//	    monthly_requests: 1500000
//	    average_duration_ms: 250
//	  aws_s3_bucket.data:
//	    standard_storage_gb: 500
//	    monthly_tier_1_requests: 1000
//
// Resource keys are Terraform addresses: `aws_instance.web`,
// `aws_instance.web[0]`, `module.frontend.aws_instance.web`. The form
// c3x printed before 0.3.19, with the kind in front of the module path
// (`aws_instance.module.frontend.web`), is still accepted.
package usage

import (
	"fmt"
	"os"
	"sort"

	"github.com/c3xdev/c3x/internal/domain"
	"gopkg.in/yaml.v3"
)

// CurrentSchemaVersion is the version string we accept today. The
// loader rejects unrecognised versions so an old c3x reading a new
// usage file with a structurally-incompatible schema gets a clear
// diagnostic instead of silently mis-merging.
const CurrentSchemaVersion = "0.1"

// File is the on-disk shape of a usage YAML.
type File struct {
	Version       string                    `yaml:"version"`
	ResourceUsage map[string]map[string]any `yaml:"resource_usage"`
	// Defaults applies its values to every resource whose attribute is
	// missing. Useful for blanket assumptions like
	// `monthly_data_processed_gb: 100` across every NAT gateway.
	Defaults map[string]map[string]any `yaml:"defaults"`
}

// Load reads `path` and returns the parsed file. Empty path is not an
// error — it just produces an empty File so callers can use
// "if usage.Load(...) was empty, do nothing" naturally.
func Load(path string) (File, error) {
	if path == "" {
		return File{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if f.Version != "" && f.Version != CurrentSchemaVersion {
		return File{}, fmt.Errorf("%s: unsupported usage schema version %q (want %q)",
			path, f.Version, CurrentSchemaVersion)
	}
	return f, nil
}

// Apply merges the usage file's quantities into every resource in
// place. Lookup is by `kind.name`; entries that don't match any
// resource are returned in `unmatched` so the caller can surface a
// warning. Defaults apply to every resource sharing the kind.
//
// Precedence: usage-file values WIN over parser-supplied attributes.
// The usage file documents what the workload actually does at
// runtime, which is more authoritative than the static default in
// the Terraform source.
func Apply(resources []domain.Resource, f File) (unmatched []string) {
	return ApplyWithReport(resources, f).Unmatched
}

// Report is what [ApplyWithReport] found beyond applying the file.
type Report struct {
	// Unmatched lists keys that matched no resource.
	Unmatched []string
	// Legacy maps each key written in the pre-0.3.19 form
	// (aws_instance.module.x.web) to its Terraform address
	// (module.x.aws_instance.web), so the caller can suggest the rename.
	Legacy map[string]string
}

// ApplyWithReport is [Apply] that also reports keys written in the
// legacy form. A key in the Terraform-address form wins when a file has
// both for the same resource.
func ApplyWithReport(resources []domain.Resource, f File) Report {
	var rep Report
	if len(f.ResourceUsage) == 0 && len(f.Defaults) == 0 {
		return rep
	}
	matched := make(map[string]struct{}, len(f.ResourceUsage))

	for i, r := range resources {
		// Defaults first (by kind), so explicit entries override them.
		if d, ok := f.Defaults[r.Ref.Kind]; ok {
			applyAttrs(&resources[i], d)
		}
		// Match by the resource's Terraform address, including module
		// prefixes and count/for_each keys, then by the legacy form.
		addr, legacy := r.Ref.Label(), r.Ref.LegacyLabel()
		if u, ok := f.ResourceUsage[addr]; ok {
			applyAttrs(&resources[i], u)
			matched[addr] = struct{}{}
		} else if u, ok := f.ResourceUsage[legacy]; ok && legacy != addr {
			applyAttrs(&resources[i], u)
			matched[legacy] = struct{}{}
			if rep.Legacy == nil {
				rep.Legacy = map[string]string{}
			}
			rep.Legacy[legacy] = addr
		}
	}
	for key := range f.ResourceUsage {
		if _, ok := matched[key]; !ok {
			rep.Unmatched = append(rep.Unmatched, key)
		}
	}
	sort.Strings(rep.Unmatched)
	return rep
}

// applyAttrs writes each key/value into the resource's Attributes map.
// Existing entries are overwritten — that's the precedence rule.
func applyAttrs(r *domain.Resource, attrs map[string]any) {
	if r.Attributes == nil {
		r.Attributes = make(map[string]any, len(attrs))
	}
	for k, v := range attrs {
		r.Attributes[k] = v
	}
}
