package terraform_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// TestFileFunctionCannotReadServerFiles is a security invariant: c3x's HCL
// evaluator deliberately does not register file()/templatefile()/etc., so a
// crafted .tf cannot exfiltrate server files or env vars (e.g. an API key in
// /proc/self/environ) through attribute evaluation. This locks that in.
func TestFileFunctionCannotReadServerFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const secret = "SUPERSECRET-DO-NOT-LEAK"
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "main.tf", `
		resource "aws_s3_bucket" "evil" {
		  bucket = file("secret.txt")
		}
	`)

	got, err := terraform.ParseDirectory(dir, terraform.Options{})
	if err != nil {
		t.Fatalf("parse errored (expected graceful degradation): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}
	if v := fmt.Sprint(got[0].Attributes["bucket"]); v == secret {
		t.Fatalf("SECURITY: file() leaked the file contents into an attribute: %q", v)
	}
}

// TestRemoteModuleNotFetchedWhenOffline is a security invariant: with module
// fetching disabled (Options.Offline — mapped from --no-remote-modules for
// untrusted input), a malicious `module { source = "git::https://…" }` must
// not trigger a git/HTTP fetch (SSRF, e.g. to the cloud metadata endpoint).
// Top-level resources still parse.
func TestRemoteModuleNotFetchedWhenOffline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "main.tf", `
		resource "aws_instance" "web" {
		  instance_type = "t3.micro"
		}
		module "pwn" {
		  source = "git::https://169.254.169.254/latest/meta-data"
		}
	`)

	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatalf("parse errored: %v", err)
	}
	var haveWeb bool
	for _, r := range got {
		if r.Ref.Kind == "aws_instance" && r.Ref.Name == "web" {
			haveWeb = true
		}
	}
	if !haveWeb {
		t.Fatalf("expected the top-level aws_instance.web to parse; got %d resources", len(got))
	}
}
