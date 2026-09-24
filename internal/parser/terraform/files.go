package terraform

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// configFiles lists the configuration files OpenTofu or Terraform would
// load from dir: every *.tf, plus OpenTofu's *.tofu.
//
// OpenTofu (1.8+) gives a .tofu file precedence over the .tf file of the
// same name, and ignores that .tf file entirely. Projects use this to run
// one codebase under both tools: main.tf for Terraform, main.tofu with
// the OpenTofu-only parts. Loading both would count every resource twice,
// so a .tf shadowed by a .tofu is dropped here, as OpenTofu drops it.
func configFiles(dir string) ([]string, error) {
	tf, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	tofu, err := filepath.Glob(filepath.Join(dir, "*.tofu"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	shadowed := make(map[string]bool, len(tofu))
	for _, p := range tofu {
		shadowed[strings.TrimSuffix(p, ".tofu")] = true
	}
	files := append([]string{}, tofu...)
	for _, p := range tf {
		if !shadowed[strings.TrimSuffix(p, ".tf")] {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}
