package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEstimateOnEmptyDirReportsClearly is the regression sentinel:
// running against a directory with no .tf files must surface that
// fact, not silently produce a $0 estimate.
func TestEstimateOnEmptyDirReportsClearly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", t.TempDir()})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error on empty dir, got nil")
	}
}

// TestEstimateAgainstRealTerraform exercises the full Phase 3 path:
// parser → catalog → calculator → text print. Pricing uses the stub,
// so subtotals are zero but the resource MUST be parsed and reach the
// engine — that's the proof Phase 3 is wired.
func TestEstimateAgainstRealTerraform(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		variable "instance_type" { default = "m5.xlarge" }
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "web" {
		  instance_type = var.instance_type
		  ami           = "ami-x"
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	got := out.String()
	// Either we see priced output (stub matched the canonical query —
	// it won't in default config) or we see the "parsed, none priced"
	// fallback. Either way the resource must have flowed through.
	if !strings.Contains(got, "resources parsed") && !strings.Contains(got, "aws_instance.web") {
		t.Errorf("expected resource flow indicator, got:\n%s", got)
	}
}

// TestEstimateWarnsOnUnpricedResource is the regression guard for #48:
// a resource that is parsed but matches no pricing dimension must be
// surfaced with a warning by default, not silently dropped from the
// output (which would let users assume it's free).
func TestEstimateWarnsOnUnpricedResource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	// db-n1-standard-1 is a legacy Cloud SQL tier with no catalog
	// mapping, so every compute dimension guards out and the instance
	// cannot be priced. The aws_instance keeps the estimate non-empty.
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "web" {
		  ami           = "ami-x"
		  instance_type = "t3.micro"
		}
		resource "google_sql_database_instance" "legacy" {
		  database_version = "POSTGRES_15"
		  region           = "us-central1"
		  settings {
		    tier = "db-n1-standard-1"
		  }
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "not priced") {
		t.Errorf("expected an unpriced-resource warning by default, got:\n%s", got)
	}
}

// TestEstimateAcceptsVarFlag wires up --var=name=value end-to-end.
func TestEstimateAcceptsVarFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		variable "env" { default = "dev" }
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "x" {
		  instance_type = "t3.micro"
		  ami           = "ami-${var.env}"
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir, "--var", `env="prod"`})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
}

// TestEstimateRespectsFormatFlag verifies the --format flag is
// accepted by the validator; the renderer responds to it in Phase 5.
func TestEstimateRespectsFormatFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		resource "aws_instance" "x" {
		  instance_type = "t3.micro"
		  ami           = "ami-x"
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir, "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
}

func TestEstimateRejectsBadFormat(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", t.TempDir(), "--format", "yaml"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}

// TestInlineDemoRunsFullPipeline exercises the catalog → expr →
// calculator → render path end-to-end from the CLI. If this passes,
// Phase 2 is wired correctly without depending on a parser.
func TestInlineDemoRunsFullPipeline(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", t.TempDir(), "--inline-demo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("inline-demo failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"aws_instance.demo",
		"PROJECT TOTAL",
		"$140.16/mo", // 730 × 0.192
	} {
		if !strings.Contains(got, want) {
			t.Errorf("inline-demo output missing %q:\n%s", want, got)
		}
	}
}
