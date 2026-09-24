package terraform_test

import (
	"slices"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// A reference to another resource's attribute resolves when that
// attribute is a literal in the same module, and stays unresolved (nil,
// reported) otherwise.
func TestResourceAttributeReferences(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		variable "type" { default = "c5.large" }
		resource "aws_instance" "base" {
		  instance_type = "m5.2xlarge"
		  from_var      = var.type
		  root_block_device { volume_size = 80 }
		}
		resource "aws_instance" "counted" {
		  count         = 1
		  instance_type = "r5.large"
		}
		resource "aws_instance" "twin" {
		  instance_type = aws_instance.base.instance_type
		  from_var      = aws_instance.base.from_var
		  ami           = aws_instance.base.id
		  other         = aws_instance.counted[0].instance_type
		  guarded       = try(aws_instance.base.instance_type, "fallback")
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	twin := byName(got)["twin"]
	if twin.Attributes["instance_type"] != "m5.2xlarge" || twin.Attributes["guarded"] != "m5.2xlarge" {
		t.Errorf("instance_type = %v, guarded = %v; want the literal m5.2xlarge",
			twin.Attributes["instance_type"], twin.Attributes["guarded"])
	}
	if slices.Contains(twin.Unresolved, "instance_type") {
		t.Errorf("instance_type reported unresolved: %v", twin.Unresolved)
	}
	// Not literal (a variable), only known after apply (id), or on a
	// counted resource: left unresolved, as before.
	for _, attr := range []string{"from_var", "ami", "other"} {
		if twin.Attributes[attr] != nil || !slices.Contains(twin.Unresolved, attr) {
			t.Errorf("%s = %v, unresolved %v; want nil and reported", attr, twin.Attributes[attr], twin.Unresolved)
		}
	}
}

// References stay within a module: a child module's resource of the same
// address is not visible to the root, and vice versa.
func TestResourceReferencesAreModuleScoped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustMkdir(t, dir+"/child")
	write(t, dir+"/child", "main.tf", `
		resource "aws_instance" "base" { instance_type = "t3.nano" }
		resource "aws_instance" "ref" { instance_type = aws_instance.base.instance_type }
	`)
	write(t, dir, "main.tf", `
		module "child" { source = "./child" }
		resource "aws_instance" "base" { instance_type = "m5.large" }
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if it := byName(got)["module.child.ref"].Attributes["instance_type"]; it != "t3.nano" {
		t.Errorf("module.child.ref instance_type = %v; want its own module's t3.nano", it)
	}
}
