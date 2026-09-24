package terraform_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// writeWebModule writes a ./web module with one instance whose type comes
// from an input, under dir.
func writeWebModule(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "web"), "main.tf", `
		variable "type" { default = "t3.micro" }
		resource "aws_instance" "this" { instance_type = var.type }
	`)
}

func names(rs []domain.Resource) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Ref.Name
	}
	sort.Strings(out)
	return out
}

// A module with count = 0 creates nothing. It used to be priced anyway,
// because count on a module block was ignored.
func TestModuleCountZeroIsNotPriced(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeWebModule(t, dir)
	write(t, dir, "main.tf", `
		variable "enabled" { default = false }
		module "web" {
		  source = "./web"
		  count  = var.enabled ? 1 : 0
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v; want no resources from a count = 0 module", names(got))
	}
}

// count expands a module into instances addressed module.web[i], and
// count.index is in scope for the instance's inputs.
func TestModuleCountExpandsInstances(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeWebModule(t, dir)
	write(t, dir, "main.tf", `
		locals { types = ["m5.large", "m5.xlarge"] }
		module "web" {
		  source = "./web"
		  count  = 2
		  type   = local.types[count.index]
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	rs := byName(got)
	if len(got) != 2 {
		t.Fatalf("got %v; want module.web[0].this and module.web[1].this", names(got))
	}
	for name, want := range map[string]string{"module.web[0].this": "m5.large", "module.web[1].this": "m5.xlarge"} {
		if it := rs[name].Attributes["instance_type"]; it != want {
			t.Errorf("%s instance_type = %v, want %s", name, it, want)
		}
	}
}

// for_each over three items prices three instances, addressed by key,
// with each.key / each.value in scope. It used to price one.
func TestModuleForEachExpandsInstances(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeWebModule(t, dir)
	write(t, dir, "main.tf", `
		module "web" {
		  source   = "./web"
		  for_each = { a = "m5.large", b = "m5.xlarge", c = "m5.2xlarge" }
		  type     = each.value
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %v; want three instances", names(got))
	}
	rs := byName(got)
	for name, want := range map[string]string{
		`module.web["a"].this`: "m5.large",
		`module.web["b"].this`: "m5.xlarge",
		`module.web["c"].this`: "m5.2xlarge",
	} {
		if it := rs[name].Attributes["instance_type"]; it != want {
			t.Errorf("%s instance_type = %v, want %s", name, it, want)
		}
	}
}

// Nested modules under a counted module carry the instance key in their
// address, and resources with their own count inside a module instance
// expand too.
func TestNestedModuleUnderForEachInstance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, d := range []string{"outer", filepath.Join("outer", "inner")} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, dir, "main.tf", `
		module "outer" {
		  source   = "./outer"
		  for_each = toset(["x", "y"])
		  size     = each.key == "x" ? 1 : 2
		}
	`)
	write(t, filepath.Join(dir, "outer"), "main.tf", `
		variable "size" {}
		module "inner" {
		  source = "./inner"
		  n      = var.size
		}
	`)
	write(t, filepath.Join(dir, "outer", "inner"), "main.tf", `
		variable "n" {}
		resource "aws_instance" "node" {
		  count         = var.n
		  instance_type = "t3.micro"
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`module.outer["x"].module.inner.node[0]`,
		`module.outer["y"].module.inner.node[0]`,
		`module.outer["y"].module.inner.node[1]`,
	}
	if got := names(got); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("got %v, want %v", got, want)
	}
}
