package terraform_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// writeTree writes files (relative path → body) under dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Dir(p), filepath.Base(p), body)
	}
}

func parseOffline(t *testing.T, dir string) map[string]map[string]any {
	t.Helper()
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]any{}
	for name, r := range byName(got) {
		out[name] = r.Attributes
	}
	return out
}

// The reproduction from c3x#116: a module's outputs feed another module's
// inputs (a for_each and an attribute) and a root resource's attribute.
// The queues used to be dropped and both instance classes priced at the
// catalog default.
func TestModuleOutputsFeedInputsAndResources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"mods/sizes/main.tf": `
			locals {
			  sizes = { prod = { class = "db.m7g.large" } }
			}
			output "sizes"  { value = local.sizes }
			output "queues" { value = ["a", "b"] }
		`,
		"mods/db/main.tf": `
			variable "class"  { type = string }
			variable "queues" { type = list(string) }
			resource "aws_db_instance" "x" {
			  engine            = "postgres"
			  instance_class    = var.class
			  allocated_storage = 50
			}
			resource "aws_sqs_queue" "q" {
			  for_each = toset(var.queues)
			  name     = each.key
			}
		`,
		"root/main.tf": `
			provider "aws" { region = "eu-west-1" }
			module "sizes" { source = "../mods/sizes" }
			module "db" {
			  source = "../mods/db"
			  class  = module.sizes.sizes["prod"].class
			  queues = module.sizes.queues
			}
			resource "aws_db_instance" "direct" {
			  engine            = "postgres"
			  instance_class    = module.sizes.sizes["prod"].class
			  allocated_storage = 50
			}
		`,
	})
	rs := parseOffline(t, filepath.Join(dir, "root"))
	for _, name := range []string{"direct", "module.db.x"} {
		if got := rs[name]["instance_class"]; got != "db.m7g.large" {
			t.Errorf("%s instance_class = %v, want db.m7g.large", name, got)
		}
	}
	for _, name := range []string{`module.db.q["a"]`, `module.db.q["b"]`} {
		if _, ok := rs[name]; !ok {
			t.Errorf("%s missing; got %v", name, keys(rs))
		}
	}
}

// A call is resolved after the modules it reads even when it is written
// first, and a local built from a module output is in scope for both
// module inputs and resources.
func TestModuleOutputsThroughLocalsInAnyOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"main.tf": `
			module "web" {
			  source = "./web"
			  type   = local.type
			}
			locals { type = module.cells.types.web }
			resource "aws_instance" "bastion" {
			  instance_type = local.type
			}
			module "cells" { source = "./cells" }
		`,
		"cells/main.tf": `
			variable "web" { default = "m5.large" }
			output "types" { value = { web = var.web } }
		`,
		"web/main.tf": `
			variable "type" {}
			resource "aws_instance" "this" { instance_type = var.type }
		`,
	})
	rs := parseOffline(t, dir)
	for _, name := range []string{"bastion", "module.web.this"} {
		if got := rs[name]["instance_type"]; got != "m5.large" {
			t.Errorf("%s instance_type = %v, want m5.large", name, got)
		}
	}
}

// Outputs of counted and for_each modules are addressed as Terraform
// addresses them: module.x[0].out and module.x["key"].out. Outputs of a
// nested module reach the root through the module in between.
func TestModuleOutputsOfInstancesAndNestedModules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"main.tf": `
			module "counted" {
			  source = "./size"
			  count  = 2
			  size   = count.index == 0 ? "t3.small" : "t3.large"
			}
			module "keyed" {
			  source   = "./size"
			  for_each = toset(["x"])
			  size     = "m5.${each.key}large"
			}
			module "outer" { source = "./outer" }
			resource "aws_instance" "a" { instance_type = module.counted[1].size }
			resource "aws_instance" "b" { instance_type = module.keyed["x"].size }
			resource "aws_instance" "c" { instance_type = module.outer.size }
		`,
		"size/main.tf": `
			variable "size" {}
			output "size" { value = var.size }
		`,
		"outer/main.tf": `
			module "inner" {
			  source = "../size"
			  size   = "c5.large"
			}
			output "size" { value = module.inner.size }
		`,
	})
	rs := parseOffline(t, dir)
	for name, want := range map[string]string{"a": "t3.large", "b": "m5.xlarge", "c": "c5.large"} {
		if got := rs[name]["instance_type"]; got != want {
			t.Errorf("%s instance_type = %v, want %s", name, got, want)
		}
	}
}

// A count from a module output expands the resource, and outputs work
// from Terraform JSON modules too.
func TestModuleOutputsFromJSONDriveCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"main.tf": `
			module "cfg" { source = "./cfg" }
			resource "aws_instance" "n" {
			  count         = module.cfg.nodes
			  instance_type = "t3.micro"
			}
		`,
		"cfg/main.tf.json": `{
			"output": { "nodes": { "value": 3, "description": "node count", "sensitive": false } }
		}`,
	})
	rs := parseOffline(t, dir)
	if got := keys(rs); !slices.Equal(got, []string{"n[0]", "n[1]", "n[2]"}) {
		t.Fatalf("got %v, want three instances of n", got)
	}
}

// Two modules reading each other's outputs can't both resolve; the parse
// finishes, and what doesn't resolve is reported as before.
func TestModuleOutputCycleTerminates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"main.tf": `
			module "a" {
			  source = "./echo"
			  in     = module.b.out
			}
			module "b" {
			  source = "./echo"
			  in     = module.a.out
			}
		`,
		"echo/main.tf": `
			variable "in" { default = "t3.micro" }
			output "out" { value = var.in }
			resource "aws_instance" "this" { instance_type = var.in }
		`,
	})
	rs := parseOffline(t, dir)
	if len(rs) != 2 {
		t.Fatalf("got %v, want module.a.this and module.b.this", keys(rs))
	}
}

func keys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
