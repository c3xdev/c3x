package terraform_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// An override file merges into the block it overrides. It used to load as
// an ordinary file, so the overridden resource was priced twice.
func TestOverrideFileMergesIntoOriginal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		resource "aws_instance" "web" {
		  instance_type = "t3.micro"
		  ami           = "ami-123"
		  ebs_block_device { volume_size = 10 }
		  ebs_block_device { volume_size = 20 }
		}
	`)
	write(t, dir, "override.tf", `
		resource "aws_instance" "web" {
		  instance_type = "m5.4xlarge"
		  ebs_block_device { volume_size = 500 }
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d resources; want the one overridden aws_instance.web", len(got))
	}
	a := got[0].Attributes
	if a["instance_type"] != "m5.4xlarge" {
		t.Errorf("instance_type = %v; the override's attribute must replace the original", a["instance_type"])
	}
	if a["ami"] != "ami-123" {
		t.Errorf("ami = %v; attributes the override doesn't set are kept", a["ami"])
	}
	// Nested blocks are replaced wholesale: both originals go.
	if want := map[string]any{"volume_size": float64(500)}; !reflect.DeepEqual(a["ebs_block_device"], want) {
		t.Errorf("ebs_block_device = %#v, want %#v", a["ebs_block_device"], want)
	}
}

// *_override files, locals, variables, data and modules follow the same
// rules, and later override files (lexical order) win.
func TestOverrideFileKinds(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "app"))
	write(t, filepath.Join(dir, "app"), "main.tf", `
		variable "n" { default = 1 }
		resource "aws_instance" "app" {
		  count         = var.n
		  instance_type = "t3.micro"
		}
	`)
	write(t, dir, "main.tf", `
		variable "size" { default = 10 }
		locals {
		  type  = "t3.micro"
		  other = "kept"
		}
		module "app" {
		  source = "./app"
		  n      = 1
		}
		resource "aws_instance" "web" {
		  instance_type = local.type
		  tags          = { other = local.other }
		  root_block_device { volume_size = var.size }
		}
	`)
	write(t, dir, "a_override.tf", `
		variable "size" { default = 50 }
		locals { type = "m5.large" }
		module "app" { n = 3 }
	`)
	write(t, dir, "b_override.tf", `
		locals { type = "m5.xlarge" }
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	rs := byName(got)
	if len(got) != 4 {
		t.Fatalf("got %v; want web plus three module.app instances", names(got))
	}
	web := rs["web"].Attributes
	if web["instance_type"] != "m5.xlarge" {
		t.Errorf("instance_type = %v; want the last override's local, m5.xlarge", web["instance_type"])
	}
	if !reflect.DeepEqual(web["tags"], map[string]any{"other": "kept"}) {
		t.Errorf("tags = %v; locals the overrides don't mention are kept", web["tags"])
	}
	if want := map[string]any{"volume_size": float64(50)}; !reflect.DeepEqual(web["root_block_device"], want) {
		t.Errorf("root_block_device = %v; want the overridden variable default", web["root_block_device"])
	}
}

// OpenTofu's override.tofu, and override files in child modules.
func TestTofuOverrideInChildModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app := filepath.Join(dir, "app")
	if err := os.Mkdir(app, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "main.tf", `module "app" { source = "./app" }`)
	write(t, app, "main.tf", `resource "aws_db_instance" "db" { instance_class = "db.t3.micro" }`)
	write(t, app, "db_override.tofu", `resource "aws_db_instance" "db" { instance_class = "db.r6g.large" }`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Attributes["instance_class"] != "db.r6g.large" {
		t.Fatalf("got %d resources, instance_class %v; want one, overridden to db.r6g.large",
			len(got), got[0].Attributes["instance_class"])
	}
}

// An override block with nothing to override is dropped, not priced as a
// new resource (Terraform rejects the configuration).
func TestOverrideWithoutOriginalIsIgnored(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `resource "aws_instance" "web" { instance_type = "t3.micro" }`)
	write(t, dir, "override.tf", `resource "aws_instance" "ghost" { instance_type = "m5.large" }`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Ref.Name != "web" {
		t.Fatalf("got %v; want only web", names(got))
	}
}
