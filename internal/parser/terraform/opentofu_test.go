package terraform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// byName indexes parsed resources by their instance name.
func byName(rs []domain.Resource) map[string]domain.Resource {
	out := map[string]domain.Resource{}
	for _, r := range rs {
		out[r.Ref.Name] = r
	}
	return out
}

func regionOf(r domain.Resource) string {
	if r.Region == nil {
		return ""
	}
	return *r.Region
}

// A main.tofu replaces main.tf of the same name, as OpenTofu does, so a
// project that ships both for the two tools is not counted twice.
func TestTofuFileShadowsTfOfSameName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `resource "aws_instance" "web" { instance_type = "m5.large" }`)
	write(t, dir, "main.tofu", `resource "aws_instance" "web" { instance_type = "m5.2xlarge" }`)
	write(t, dir, "db.tf", `resource "aws_db_instance" "db" { instance_class = "db.t3.micro" }`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resources (main.tofu + db.tf), got %d", len(got))
	}
	if it := byName(got)["web"].Attributes["instance_type"]; it != "m5.2xlarge" {
		t.Errorf("instance_type = %v, want the main.tofu value m5.2xlarge", it)
	}
}

func TestTofuOnlyDirectoryAndSingleFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tofu", `resource "aws_instance" "web" { instance_type = "m5.large" }`)
	for _, path := range []string{dir, filepath.Join(dir, "main.tofu")} {
		var got []domain.Resource
		var err error
		if path == dir {
			got, err = terraform.ParseDirectory(path, terraform.Options{})
		} else {
			got, err = terraform.ParseFile(path, terraform.Options{})
		}
		if err != nil || len(got) != 1 {
			t.Errorf("%s: got %d resources, err %v; want 1", path, len(got), err)
		}
	}
}

// Child modules follow the same rule as the root.
func TestTofuFilesInChildModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `module "app" { source = "./app" }`)
	if err := os.Mkdir(filepath.Join(dir, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "app"), "main.tf", `resource "aws_instance" "web" { instance_type = "m5.large" }`)
	write(t, filepath.Join(dir, "app"), "main.tofu", `resource "aws_instance" "web" { instance_type = "m5.2xlarge" }`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Attributes["instance_type"] != "m5.2xlarge" {
		t.Fatalf("got %d resources, instance_type %v; want 1 from app/main.tofu", len(got), got[0].Attributes["instance_type"])
	}
}

// Resources on an aliased provider are priced in that provider's region.
// Previously every resource took the first provider block's region.
func TestResourceUsesItsAliasedProviderRegion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		provider "aws" {
		  alias  = "eu"
		  region = "eu-west-1"
		}
		resource "aws_instance" "us" { instance_type = "m5.large" }
		resource "aws_instance" "eu" {
		  provider      = aws.eu
		  instance_type = "m5.large"
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rs := byName(got)
	if r := regionOf(rs["us"]); r != "us-east-1" {
		t.Errorf("us region = %q, want us-east-1", r)
	}
	if r := regionOf(rs["eu"]); r != "eu-west-1" {
		t.Errorf("eu region = %q, want eu-west-1", r)
	}
}

// The alias order in the file must not matter.
func TestAliasDeclaredBeforeDefaultProvider(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" {
		  alias  = "eu"
		  region = "eu-west-1"
		}
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "us" { instance_type = "m5.large" }
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r := regionOf(got[0]); r != "us-east-1" {
		t.Errorf("region = %q, want us-east-1 (the default provider, not the alias)", r)
	}
}

// OpenTofu 1.9 provider for_each: each resource instance resolves its
// provider instance through its own each.key.
func TestProviderForEachRegionPerInstance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tofu", `
		variable "regions" {
		  default = { use1 = "us-east-1", euw1 = "eu-west-1" }
		}
		provider "aws" {
		  alias    = "by_region"
		  for_each = var.regions
		  region   = each.value
		}
		resource "aws_instance" "web" {
		  for_each      = var.regions
		  provider      = aws.by_region[each.key]
		  instance_type = "m5.large"
		}
		resource "aws_instance" "fixed" {
		  provider      = aws.by_region["euw1"]
		  instance_type = "m5.large"
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rs := byName(got)
	for name, want := range map[string]string{`web["use1"]`: "us-east-1", `web["euw1"]`: "eu-west-1", "fixed": "eu-west-1"} {
		if r := regionOf(rs[name]); r != want {
			t.Errorf("%s region = %q, want %q", name, r, want)
		}
	}
}

// providers = { aws = aws.eu } deploys a module into the aliased region;
// without a mapping a child inherits the parent's default provider.
func TestModuleProvidersMapping(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		provider "aws" {
		  alias  = "eu"
		  region = "eu-west-1"
		}
		module "us" { source = "./web" }
		module "eu" {
		  source    = "./web"
		  providers = { aws = aws.eu }
		}
	`)
	if err := os.Mkdir(filepath.Join(dir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "web"), "main.tf", `resource "aws_instance" "web" { instance_type = "m5.large" }`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	rs := byName(got)
	if r := regionOf(rs["module.us.web"]); r != "us-east-1" {
		t.Errorf("module.us region = %q, want us-east-1", r)
	}
	if r := regionOf(rs["module.eu.web"]); r != "eu-west-1" {
		t.Errorf("module.eu region = %q, want eu-west-1", r)
	}
}

// An unsupported function in a local used to drop every resource reading
// it. cidrsubnet is now implemented, so all three NAT gateways appear.
func TestFunctionInLocalNoLongerDropsResources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		locals {
		  azs     = ["a", "b", "c"]
		  subnets = { for i, az in local.azs : az => cidrsubnet("10.0.0.0/16", 8, i) }
		}
		resource "aws_nat_gateway" "nat" {
		  for_each  = local.subnets
		  subnet_id = each.value
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d NAT gateways, want 3", len(got))
	}
	if s := byName(got)[`nat["c"]`].Attributes["subnet_id"]; s != "10.0.2.0/24" {
		t.Errorf(`nat["c"].subnet_id = %v, want 10.0.2.0/24`, s)
	}
}
