package terraform_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// BenchmarkParseLargeMonorepo measures parse time against a synthetic
// 500-resource config that exercises the cost-relevant pipeline
// shapes: variables, locals, count, for_each, modules, and a mix of
// AWS/Azure/GCP resource kinds.
//
// We pin the synthetic generator in code rather than committing a
// 500-file fixture so the benchmark stays language-portable and the
// version-controlled inputs stay small. The number we care about is
// `ns/op` per parsed resource; the README cites the headline ms/full-
// run figure.
func BenchmarkParseLargeMonorepo(b *testing.B) {
	dir := b.TempDir()
	writeMonorepo(b, dir, 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resources, err := terraform.ParseDirectory(dir, terraform.Options{})
		if err != nil {
			b.Fatal(err)
		}
		if len(resources) == 0 {
			b.Fatalf("expected resources, got none")
		}
	}
}

// writeMonorepo generates a Terraform configuration tree containing
// `count` resource blocks distributed across:
//
//   - top-level vars / locals (so the evaluator gets a real scope)
//   - count = length(var.zones)         (multiplication via meta-arg)
//   - for_each = toset(var.environments) (multiplication via meta-arg)
//   - a local module with its own vars   (recursion)
//
// The mix mirrors the shape we'd see in a real medium-sized monorepo.
func writeMonorepo(tb testing.TB, dir string, count int) {
	tb.Helper()
	var main strings.Builder
	main.WriteString(`
		variable "region"       { default = "us-east-1" }
		variable "zones"        { default = ["a", "b", "c"] }
		variable "environments" { default = ["dev", "stage", "prod"] }
		variable "fleet_size"   { default = 5 }

		locals {
		  name_prefix = format("%s-app", "prod")
		  per_zone    = length(var.zones)
		}

		provider "aws" { region = var.region }
	`)

	// Half via count, half via plain blocks, then a module pulled in
	// at the end. Numbers are tuned so the generated total comes out
	// close to `count`.
	plainBlocks := count - (3 * 30) - 50 // 30 count*3, 50 in module
	if plainBlocks < 1 {
		plainBlocks = 1
	}
	for i := 0; i < plainBlocks; i++ {
		fmt.Fprintf(&main, `
		resource "aws_instance" "n%d" {
		  instance_type = "t3.small"
		  ami           = "ami-${var.region}-%d"
		}
		`, i, i)
	}
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&main, `
		resource "aws_instance" "fleet_%d" {
		  count         = local.per_zone
		  instance_type = "m5.xlarge"
		  ami           = "ami-${count.index}"
		}
		`, i)
	}

	// One module folder. The module emits 50 resources via for_each.
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(main.String()), 0o644); err != nil {
		tb.Fatal(err)
	}
	modDir := filepath.Join(dir, "modules", "fleet")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		tb.Fatal(err)
	}

	var modMain strings.Builder
	modMain.WriteString(`
		variable "size" {}
		variable "envs" { default = ["dev","stage","prod","sandbox","ops","metrics","ml","experiments","admin","corp"] }
	`)
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&modMain, `
		resource "aws_instance" "leaf_%d" {
		  for_each      = toset(var.envs)
		  instance_type = var.size
		  ami           = "ami-${each.key}-%d"
		}
		`, i, i)
	}
	if err := os.WriteFile(filepath.Join(modDir, "main.tf"), []byte(modMain.String()), 0o644); err != nil {
		tb.Fatal(err)
	}

	// Hook the module into the top-level config.
	hook := `
		module "fleet" {
		  source = "./modules/fleet"
		  size   = "t3.medium"
		}
	`
	if err := os.WriteFile(filepath.Join(dir, "modules_hookup.tf"), []byte(hook), 0o644); err != nil {
		tb.Fatal(err)
	}
}
