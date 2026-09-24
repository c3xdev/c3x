package terraform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// Config-driven fleets: yamldecode(file(...)) and fileset feeding
// for_each, and a module reading its own file through path.module.
// Before these functions existed every one of these resources was either
// dropped or priced at its default.
func TestFileFunctionsFeedForEachAndModules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "fleet.yaml", "web: m5.large\nworker: m5.2xlarge\n")
	mustMkdir(t, filepath.Join(dir, "configs", "team"))
	write(t, filepath.Join(dir, "configs", "team"), "api.yaml", "size: m5.large\n")
	write(t, filepath.Join(dir, "configs"), "README.txt", "not a config\n")
	mustMkdir(t, filepath.Join(dir, "svc"))
	write(t, filepath.Join(dir, "svc"), "sizes.json", `{"size":"r5.large"}`)
	write(t, filepath.Join(dir, "svc"), "main.tf", `
		locals { cfg = jsondecode(file("${path.module}/sizes.json")) }
		resource "aws_db_instance" "db" { instance_class = "db.${local.cfg.size}" }
	`)
	write(t, dir, "size.tpl", "m5.${size}")
	write(t, dir, "main.tf", `
		locals {
		  fleet   = yamldecode(file("${path.module}/fleet.yaml"))
		  configs = { for f in fileset("${path.module}/configs", "**/*.yaml") : f => yamldecode(file("${path.module}/configs/${f}")) }
		}
		resource "aws_instance" "fleet" {
		  for_each      = local.fleet
		  instance_type = each.value
		}
		resource "aws_instance" "team" {
		  for_each      = local.configs
		  instance_type = each.value.size
		}
		resource "aws_instance" "tpl" { instance_type = templatefile("${path.module}/size.tpl", { size = "xlarge" }) }
		module "svc" { source = "./svc" }
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{AllowFileFunctions: true})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		`fleet["web"]`:          "m5.large",
		`fleet["worker"]`:       "m5.2xlarge",
		`team["team/api.yaml"]`: "m5.large", // README.txt must not match **/*.yaml
		"tpl":                   "m5.xlarge",
	}
	rs := byName(got)
	for name, it := range want {
		if v := rs[name].Attributes["instance_type"]; v != it {
			t.Errorf("%s instance_type = %v, want %s", name, v, it)
		}
	}
	if v := rs["module.svc.db"].Attributes["instance_class"]; v != "db.r5.large" {
		t.Errorf("module.svc.db instance_class = %v, want db.r5.large (path.module must be the module's own dir)", v)
	}
	if len(got) != len(want)+1 {
		t.Errorf("got %d resources, want %d", len(got), len(want)+1)
	}
}

// With file functions allowed, reads must still stay inside the project:
// a user may opt in for a repository whose CI also runs untrusted pull
// requests. The allowed read is the positive control: it proves
// file contents do reach attributes, so the refused ones being empty
// means they were refused, not merely not surfaced.
func TestFileFunctionsCannotReadOutsideProject(t *testing.T) {
	t.Parallel()
	outer := t.TempDir()
	write(t, outer, "secret.txt", "TOP-SECRET")
	dir := filepath.Join(outer, "proj")
	mustMkdir(t, dir)
	if err := os.Symlink(filepath.Join(outer, "secret.txt"), filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "inside.txt", "INSIDE")
	write(t, dir, "main.tf", `
		resource "aws_instance" "web" {
		  instance_type = "m5.large"
		  # One attribute per read: an object literal fails as a whole if
		  # any element errors, which would hide the positive control.
		  inside   = file("${path.module}/inside.txt")
		  absolute = file("`+filepath.ToSlash(filepath.Join(outer, "secret.txt"))+`")
		  dotdot   = file("${path.module}/../secret.txt")
		  symlink  = file("${path.module}/link.txt")
		  walk     = join(",", fileset("${path.module}/..", "*"))
		  probe    = tostring(fileexists("${path.module}/../secret.txt"))
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{AllowFileFunctions: true})
	if err != nil {
		t.Fatal(err)
	}
	attrs := got[0].Attributes
	if attrs["inside"] != "INSIDE" {
		t.Fatalf("positive control failed: inside = %v, want INSIDE", attrs["inside"])
	}
	for _, k := range []string{"absolute", "dotdot", "symlink", "walk", "probe"} {
		if v, ok := attrs[k]; ok && v != nil {
			t.Errorf("%s = %v: a read outside the project must be refused", k, v)
		}
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// Off by default: nothing is read unless the caller opts in. The original
// invariant (security_test.go) covers a file inside the project; this
// covers the configuration-driven pattern the functions exist for.
func TestFileFunctionsOffByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "fleet.yaml", "web: m5.large\n")
	write(t, dir, "main.tf", `
		locals { fleet = yamldecode(file("${path.module}/fleet.yaml")) }
		resource "aws_instance" "fleet" {
		  for_each      = local.fleet
		  instance_type = each.value
		}
		resource "aws_instance" "fixed" { instance_type = "m5.large" }
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Ref.Name != "fixed" {
		t.Fatalf("got %d resources; with file functions off only the fixed one should parse", len(got))
	}
}
