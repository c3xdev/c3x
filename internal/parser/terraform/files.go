package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// configExtensions are the file suffixes OpenTofu and Terraform load as
// configuration, each paired with the Terraform suffix an OpenTofu file
// of the same name shadows (main.tofu hides main.tf, main.tofu.json hides
// main.tf.json).
var configExtensions = []struct{ ext, shadows string }{
	{ext: ".tf"},
	{ext: ".tf.json"},
	{ext: ".tofu", shadows: ".tf"},
	{ext: ".tofu.json", shadows: ".tf.json"},
}

// configFiles lists the configuration files OpenTofu or Terraform would
// load from dir: every *.tf and *.tf.json, plus OpenTofu's *.tofu and
// *.tofu.json.
//
// OpenTofu (1.8+) gives a .tofu file precedence over the .tf file of the
// same name, and ignores that .tf file entirely. Projects use this to run
// one codebase under both tools: main.tf for Terraform, main.tofu with
// the OpenTofu-only parts. Loading both would count every resource twice,
// so a .tf shadowed by a .tofu is dropped here, as OpenTofu drops it.
//
// The result includes override files; splitOverrides separates them.
func configFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	// One directory read, not a glob per extension: module expansion
	// lists every module directory once per instance.
	var candidates []string
	shadowed := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		for _, ce := range configExtensions {
			if strings.HasSuffix(name, ce.ext) && len(name) > len(ce.ext) {
				p := filepath.Join(dir, name)
				candidates = append(candidates, p)
				if ce.shadows != "" {
					shadowed[strings.TrimSuffix(p, ce.ext)+ce.shadows] = true
				}
				break
			}
		}
	}
	var files []string
	for _, p := range candidates {
		if !shadowed[p] {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}

// isOverrideFile reports whether path is a Terraform override file:
// override.tf, override.tf.json, or any name ending in _override.tf /
// _override.tf.json, plus the OpenTofu .tofu / .tofu.json equivalents.
// Override files are merged into the blocks they override instead of
// being loaded as ordinary configuration (see mergeOverrides).
func isOverrideFile(path string) bool {
	base := filepath.Base(path)
	for _, ext := range []string{".tofu.json", ".tf.json", ".tofu", ".tf"} {
		if stem, ok := strings.CutSuffix(base, ext); ok {
			return stem == "override" || strings.HasSuffix(stem, "_override")
		}
	}
	return false
}

// isJSONConfig reports whether path uses Terraform's JSON syntax.
func isJSONConfig(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".tf.json") || strings.HasSuffix(lower, ".tofu.json")
}

// splitOverrides separates override files from primary ones, keeping
// both in lexical order, which is the order Terraform applies overrides.
func splitOverrides(paths []string) (primary, overrides []string) {
	for _, p := range paths {
		if isOverrideFile(p) {
			overrides = append(overrides, p)
		} else {
			primary = append(primary, p)
		}
	}
	return primary, overrides
}
