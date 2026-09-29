package terraform_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/terraform"
)

func parseWithLogs(t *testing.T, dir string) ([]domain.Resource, string) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	return got, logs.String()
}

// The one-NAT-gateway-per-AZ pattern. count read a data source attribute
// that didn't exist, so the whole resource vanished from the estimate.
// Now the zones are a placeholder of three, the resource is priced three
// times, and the assumption is reported.
func TestNATGatewayPerAvailabilityZone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "eu-west-1" }
		data "aws_availability_zones" "available" { state = "available" }
		resource "aws_nat_gateway" "nat" {
		  count             = length(data.aws_availability_zones.available.names)
		  connectivity_type = "public"
		  availability_zone = data.aws_availability_zones.available.names[count.index]
		}
	`)
	got, logs := parseWithLogs(t, dir)
	if len(got) != 3 {
		t.Fatalf("got %d NAT gateways, want 3 (one per placeholder zone)", len(got))
	}
	nat := byName(got)["nat[1]"]
	if nat.Attributes["connectivity_type"] != "public" {
		t.Errorf("connectivity_type = %v; literal attributes still resolve", nat.Attributes["connectivity_type"])
	}
	// An attribute computed from a placeholder is reported unresolved,
	// exactly as before placeholders existed.
	if nat.Attributes["availability_zone"] != nil || !slices.Contains(nat.Unresolved, "availability_zone") {
		t.Errorf("availability_zone = %v, unresolved %v; want nil and reported", nat.Attributes["availability_zone"], nat.Unresolved)
	}
	if slices.Contains(nat.Unresolved, "connectivity_type") {
		t.Errorf("unresolved %v; connectivity_type doesn't depend on a placeholder", nat.Unresolved)
	}
	// The count assumption is recorded on the instance, naming the data
	// source and the value assumed, for the assumed_count caveat.
	if want := "data.aws_availability_zones.available.names = [eu-west-1a, eu-west-1b, eu-west-1c]"; nat.AssumedCount != want {
		t.Errorf("AssumedCount = %q, want %q", nat.AssumedCount, want)
	}
	// The caveat says it where the number is; the log stays quiet.
	if strings.Contains(logs, "aws_nat_gateway.nat") {
		t.Errorf("unexpected warning for the count; logs:\n%s", logs)
	}
}

// The terraform-aws-modules/vpc shape: zones sliced into a local, passed
// into a module, used as a for_each there. The placeholder is traced
// through all of it.
func TestPlaceholderTracedThroughLocalsAndModules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mod := filepath.Join(dir, "vpc")
	if err := os.Mkdir(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, mod, "main.tf", `
		variable "azs" {}
		resource "aws_subnet" "private" {
		  for_each          = toset(var.azs)
		  availability_zone = each.key
		  cidr_block        = "10.0.0.0/24"
		}
	`)
	write(t, dir, "main.tf", `
		data "aws_availability_zones" "available" {}
		locals { azs = slice(data.aws_availability_zones.available.names, 0, 2) }
		module "vpc" {
		  source = "./vpc"
		  azs    = local.azs
		}
	`)
	got, _ := parseWithLogs(t, dir)
	if len(got) != 2 {
		t.Fatalf("got %d subnets, want 2", len(got))
	}
	s := byName(got)[`module.vpc.private["us-east-1a"]`]
	if !slices.Contains(s.Unresolved, "availability_zone") || slices.Contains(s.Unresolved, "cidr_block") {
		t.Errorf("unresolved = %v; want availability_zone (from each.key) only", s.Unresolved)
	}
	if !strings.Contains(s.AssumedCount, "data.aws_availability_zones.available.names") {
		t.Errorf("AssumedCount = %q; the for_each rests on the zones placeholder", s.AssumedCount)
	}
}

// aws_region and google_client_config resolve to the provider's region,
// aws_caller_identity to a placeholder account, all reported when a
// resource's instances depend on them.
func TestRegionAndIdentityPlaceholders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "ap-southeast-2" }
		provider "google" { region = "europe-west4" }
		data "aws_region" "current" {}
		data "aws_caller_identity" "me" {}
		data "google_client_config" "cfg" {}
		resource "aws_instance" "syd" {
		  count         = data.aws_region.current.name == "ap-southeast-2" ? 1 : 0
		  instance_type = "t3.micro"
		}
		resource "aws_instance" "acct" {
		  count         = length(data.aws_caller_identity.me.account_id) == 12 ? 1 : 0
		  instance_type = "t3.micro"
		}
		resource "google_compute_instance" "vm" {
		  count        = data.google_client_config.cfg.region == "europe-west4" ? 2 : 0
		  machine_type = "e2-small"
		}
	`)
	got, _ := parseWithLogs(t, dir)
	if len(got) != 4 {
		t.Fatalf("got %v; want syd[0], acct[0], vm[0], vm[1]", names(got))
	}
	by := byName(got)
	for name, want := range map[string]string{
		"syd[0]":  "data.aws_region.current.name = ap-southeast-2",
		"acct[0]": "data.aws_caller_identity.me.account_id = ",
		"vm[1]":   "data.google_client_config.cfg.region = europe-west4",
	} {
		if !strings.HasPrefix(by[name].AssumedCount, want) {
			t.Errorf("%s: AssumedCount = %q, want it to start with %q", name, by[name].AssumedCount, want)
		}
	}
}

// Literal arguments on the data block beat placeholders, and a data source
// without placeholders behaves as before.
func TestDataLiteralsBeatPlaceholders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		data "aws_region" "other" { name = "eu-central-1" }
		resource "aws_instance" "x" {
		  for_each      = toset([data.aws_region.other.name])
		  instance_type = "t3.micro"
		}
		data "aws_ami" "ubuntu" { most_recent = true }
		resource "aws_instance" "y" {
		  count = length(data.aws_ami.ubuntu.image_id)
		}
	`)
	got, logs := parseWithLogs(t, dir)
	rs := byName(got)
	if _, ok := rs[`x["eu-central-1"]`]; !ok {
		t.Errorf("got %v; want x[\"eu-central-1\"]", names(got))
	}
	if strings.Contains(logs, "aws_instance.x") {
		t.Errorf("a literal data argument is configuration, not a placeholder; logs:\n%s", logs)
	}
	if len(got) != 1 {
		t.Errorf("got %v; aws_ami has no placeholder, so y is still omitted", names(got))
	}
}

// A count over the placeholder zones records the assumption on every
// instance, for the calculator's assumed_count caveat; a literal count in
// the same configuration records none.
func TestAssumedCountRecordedOnInstances(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		data "aws_availability_zones" "available" {}
		resource "aws_nat_gateway" "nat" {
		  count = length(data.aws_availability_zones.available.names)
		}
		resource "aws_eip" "nat" {
		  for_each = toset(data.aws_availability_zones.available.names)
		  domain   = "vpc"
		}
		resource "aws_instance" "web" {
		  count         = 2
		  instance_type = "t3.micro"
		}
		resource "aws_instance" "one" {
		  instance_type = "t3.micro"
		}
	`)
	got, _ := parseWithLogs(t, dir)
	want := "data.aws_availability_zones.available.names = [us-east-1a, us-east-1b, us-east-1c]"
	by := byName(got)
	for _, n := range []string{"nat[0]", "nat[1]", "nat[2]", `nat["us-east-1b"]`} {
		if got := by[n].AssumedCount; got != want {
			t.Errorf("%s: AssumedCount = %q, want %q", n, got, want)
		}
	}
	for _, n := range []string{"web[0]", "web[1]", "one"} {
		r, ok := by[n]
		if !ok {
			t.Fatalf("%s missing; got %v", n, names(got))
		}
		if got := r.AssumedCount; got != "" {
			t.Errorf("%s: AssumedCount = %q; a literal count assumes nothing", n, got)
		}
	}
}

// A module-level count over a placeholder makes every resource in the
// module an assumption, even one with no count of its own.
func TestAssumedCountPropagatesFromModuleCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mod := filepath.Join(dir, "zone")
	if err := os.Mkdir(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, mod, "main.tf", `
		variable "az" {}
		resource "aws_nat_gateway" "this" {}
	`)
	write(t, dir, "main.tf", `
		data "aws_availability_zones" "available" {}
		module "zone" {
		  source = "./zone"
		  count  = length(data.aws_availability_zones.available.names)
		  az     = data.aws_availability_zones.available.names[count.index]
		}
		module "fixed" {
		  source = "./zone"
		  count  = 1
		  az     = "us-east-1a"
		}
	`)
	got, _ := parseWithLogs(t, dir)
	by := byName(got)
	if len(got) != 4 {
		t.Fatalf("got %v; want three zone instances and one fixed", names(got))
	}
	if a := by["module.zone[2].this"].AssumedCount; !strings.Contains(a, "data.aws_availability_zones.available.names = [us-east-1a") {
		t.Errorf("module.zone[2].this AssumedCount = %q", a)
	}
	if r, ok := by["module.fixed[0].this"]; !ok || r.AssumedCount != "" {
		t.Errorf("module.fixed[0].this present=%v AssumedCount=%q; want present, empty", ok, r.AssumedCount)
	}
}
