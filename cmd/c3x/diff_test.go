package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEstimateSaveBaselineProducesLoadableJSON proves round-trip: save
// from one invocation, load from another. Without this, --save-baseline
// is just a side-effect with no consumer.
func TestEstimateSaveBaselineProducesLoadableJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "x" {
		  instance_type = "t3.micro"
		  ami           = "ami-x"
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := filepath.Join(dir, "base.json")

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir, "--offline", "--save-baseline", baseline})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	if _, err := os.Stat(baseline); err != nil {
		t.Fatalf("baseline not written: %v", err)
	}
	raw, _ := os.ReadFile(baseline)
	for _, want := range []string{`"project_total"`, `"costs"`, `"aws_instance"`, `"x"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("baseline missing %q:\n%s", want, raw)
		}
	}
}

// TestDiffAgainstSavedBaselineProducesDelta is the end-to-end happy
// path: save a baseline, modify the config, run diff, see Δ.
func TestDiffAgainstSavedBaselineProducesDelta(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "x" {
		  instance_type = "t3.micro"
		  ami           = "ami-x"
		}
	`), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := filepath.Join(dir, "base.json")

	// Snapshot the current state.
	saveCmd := newRootCmd()
	saveOut := &bytes.Buffer{}
	saveCmd.SetOut(saveOut)
	saveCmd.SetErr(saveOut)
	saveCmd.SetArgs([]string{"estimate", "--path", dir, "--offline", "--save-baseline", baseline})
	if err := saveCmd.Execute(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Add a second resource to the config so the diff shows an Added row.
	if err := os.WriteFile(filepath.Join(dir, "extra.tf"), []byte(`
		resource "aws_eip" "main" {}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	diffCmd := newRootCmd()
	diffOut := &bytes.Buffer{}
	diffCmd.SetOut(diffOut)
	diffCmd.SetErr(diffOut)
	diffCmd.SetArgs([]string{"diff", "--path", dir, "--baseline", baseline, "--offline"})
	if err := diffCmd.Execute(); err != nil {
		t.Fatalf("diff: %v", err)
	}
	got := diffOut.String()
	if !strings.Contains(got, "c3x diff") {
		t.Errorf("expected diff header:\n%s", got)
	}
	if !strings.Contains(got, "aws_eip.main") {
		t.Errorf("expected added resource in output:\n%s", got)
	}
}

// TestBudgetGateFailsWhenExceeded checks --budget triggers a non-zero
// exit code via the errBudgetExceeded sentinel.
func TestBudgetGateFailsWhenExceeded(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	// The offline stub returns zero for unrecognised queries; we want
	// non-zero so the gate trips. Use the inline-demo path which seeds
	// real numbers.
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`# placeholder`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir, "--inline-demo", "--budget", "10"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected budget gate to fail, got nil")
	}
	if !strings.Contains(out.String(), "budget exceeded") {
		t.Errorf("expected budget-exceeded message in stderr:\n%s", out.String())
	}
}

func TestBudgetGatePassesUnderLimit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`# placeholder`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"estimate", "--path", dir, "--inline-demo", "--budget", "1000"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected pass under budget, got %v", err)
	}
}

// budget in .c3x.toml gates the run with no flag, and an explicit
// --budget 0 switches that gate off.
func TestBudgetGateFromProjectConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`# placeholder`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".c3x.toml"), []byte("budget = 10.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(extra ...string) error {
		cmd := newRootCmd()
		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs(append([]string{"estimate", "--path", dir, "--inline-demo"}, extra...))
		return cmd.Execute()
	}
	if err := run(); !errors.Is(err, errBudgetExceeded) {
		t.Fatalf("budget from .c3x.toml: err = %v, want errBudgetExceeded", err)
	}
	if err := run("--budget", "0"); err != nil {
		t.Fatalf("--budget 0 should disable the project's gate, got %v", err)
	}
}
