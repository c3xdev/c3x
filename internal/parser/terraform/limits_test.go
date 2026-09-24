package terraform_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// Each of these ran unbounded before the parse budget existed.

func expectLimitError(t *testing.T, dir string) {
	t.Helper()
	start := time.Now()
	_, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err == nil || !strings.Contains(err.Error(), "parse limits") {
		t.Fatalf("err = %v; want a parse-limit error", err)
	}
	// Generous for CI's race detector: the point is that it terminates
	// promptly, which the budget guarantees, not a benchmark.
	if d := time.Since(start); d > 60*time.Second {
		t.Fatalf("refusing took %s; limits must trip early", d)
	}
}

func TestCountBombIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `resource "aws_instance" "x" { count = 1000000000 }`)
	expectLimitError(t, dir)
}

func TestForEachOverHugeMapIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A literal map one past the per-resource limit: cheap to parse, so
	// the test measures the limit rather than HCL's evaluation speed.
	var b strings.Builder
	b.WriteString("locals {\n  keys = {\n")
	for i := 0; i <= 10_000; i++ {
		fmt.Fprintf(&b, "    k%d = %d\n", i, i)
	}
	b.WriteString("  }\n}\nresource \"aws_instance\" \"x\" { for_each = local.keys }\n")
	write(t, dir, "main.tf", b.String())
	expectLimitError(t, dir)
}

// Four modules sourcing their own directory recurse to MaxModuleDepth,
// 4^10 expansions, without a budget across the whole parse.
func TestSelfRecursiveModulesAreRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		module "a" { source = "./" }
		module "b" { source = "./" }
		module "c" { source = "./" }
		module "d" { source = "./" }
		resource "aws_instance" "x" {}
	`)
	expectLimitError(t, dir)
}

func TestHugeRangeIsRefusedBeforeAllocating(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		locals { big = range(100000000) }
		resource "aws_instance" "x" { count = length(local.big) }
	`)
	start := time.Now()
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d instances; the oversized range must not resolve", len(got))
	}
	if time.Since(start) > 5*time.Second {
		t.Error("range(100000000) must be refused, not allocated")
	}
}

func TestGzipBombIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 17 MiB of zeros, just past the 16 MiB cap: small once compressed,
	// large inflated. The small round trip is the positive control: it
	// must resolve, so the large one failing is the cap and not the
	// expression.
	write(t, dir, "main.tf", `
		locals {
		  small = base64gunzip(base64gzip("m5.large"))
		  big   = base64gunzip(base64gzip(join("", [for i in range(1024) : format("%017408d", 0)])))
		}
		resource "aws_instance" "x" {
		  instance_type = local.small
		  big_length    = length(local.big)
		}
	`)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Attributes["instance_type"] != "m5.large" {
		t.Fatalf("positive control failed: instance_type = %v", got[0].Attributes["instance_type"])
	}
	if v := got[0].Attributes["big_length"]; v != nil {
		t.Errorf("big_length = %v; decompressing past the cap must fail", v)
	}
}

// In untrusted mode a local module may not reach outside the scanned
// directory; trusted parses keep Terraform's behaviour.
func TestUntrustedModuleSourceConfinedToProject(t *testing.T) {
	t.Parallel()
	outer := t.TempDir()
	mustMkdir(t, filepath.Join(outer, "other-upload"))
	write(t, filepath.Join(outer, "other-upload"), "main.tf", `resource "aws_db_instance" "theirs" { instance_class = "db.r5.4xlarge" }`)
	dir := filepath.Join(outer, "proj")
	mustMkdir(t, dir)
	write(t, dir, "main.tf", `
		module "peek" { source = "../other-upload" }
		resource "aws_instance" "mine" {}
	`)
	untrusted, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true, Untrusted: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(untrusted) != 1 || untrusted[0].Ref.Name != "mine" {
		t.Errorf("untrusted parse got %d resources; another directory's module must not be read", len(untrusted))
	}
	trusted, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(trusted) != 2 {
		t.Errorf("trusted parse got %d resources; ../ modules are legitimate in your own repo", len(trusted))
	}
}

// git must not clone from the host's filesystem (file://) for a module.
func TestGitModuleFromLocalFilesystemIsRefused(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.email=a@b", "-c", "user.name=a", "commit", "-q", "--allow-empty", "-m", "x"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "main.tf"), []byte(`resource "aws_instance" "host" {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := terraform.NewModuleFetcher(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if dir, err := f.Fetch("git::file://"+repo, ""); err == nil {
		t.Fatalf("fetched %s from the local filesystem; file:// must be refused", dir)
	}
}
