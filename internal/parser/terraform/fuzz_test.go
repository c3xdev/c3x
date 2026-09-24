package terraform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// FuzzParseDirectory throws arbitrary byte sequences at the parser as
// if they were main.tf and asserts that we never panic. Any input
// either parses, returns a wrapped error, or returns no resources —
// what we never accept is a goroutine death that takes the CLI down.
//
// Run locally with `go test -fuzz=FuzzParseDirectory -fuzztime=30s
// ./internal/parser/terraform/`. CI runs the seed corpus only.
func FuzzParseDirectory(f *testing.F) {
	// Seeds: representative shapes that exercised parser branches in
	// integration tests. The fuzzer mutates around these.
	seeds := []string{
		`provider "aws" { region = "us-east-1" }`,
		`variable "x" { default = "v" }`,
		`resource "aws_instance" "x" { instance_type = var.t }`,
		`locals { x = format("%s-%d", "a", 1) }`,
		`module "m" { source = "./m" }`,
		``,
		`resource "" "" {}`,
		`# only a comment`,
		`provider "aws" { region = var.region }`,
		`resource "aws_instance" "x" { count = length([1,2,3]) }`,
		`resource "aws_instance" "x" { dynamic "d" { for_each = [1] content { v = d.value } } }`,
		`data "aws_availability_zones" "a" {}
resource "aws_nat_gateway" "n" { for_each = toset(data.aws_availability_zones.a.names) }`,
		`module "m" { source = "./" count = 2 x = count.index }`,
		`resource "aws_instance" "a" { t = "x" }
resource "aws_instance" "b" { t = aws_instance.a.t }`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), raw, 0o644); err != nil {
			t.Skip(err)
		}
		// We don't care about the return values — we only care that the
		// call doesn't panic. A wrapped error or empty result is fine.
		// Offline: the fuzzer can synthesise registry/git module
		// sources; resolution must never leave the process.
		_, _ = terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	})
}

// FuzzParseJSONDirectory does the same for the JSON syntax, which is
// translated to native syntax before parsing: no input may panic the
// translator or the parse of what it emits.
func FuzzParseJSONDirectory(f *testing.F) {
	seeds := []string{
		`{}`,
		`{"resource": {"aws_instance": {"x": {"instance_type": "${var.t}"}}}}`,
		`{"variable": {"t": {"type": "list(string)", "default": ["a"]}}}`,
		`{"locals": {"a": [1, 2.5e3, null, true, {"k": "v"}]}}`,
		`{"resource": {"aws_instance": {"x": {"dynamic": {"d": {"for_each": [1], "content": {"v": "${d.value}"}}}}}}}`,
		`{"module": {"m": {"source": "./", "providers": {"aws": "aws.eu"}}}}`,
		`{"provider": {"aws": [{"region": "us-east-1"}, {"alias": "eu"}]}}`,
		`[1]`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.tf.json"), raw, 0o644); err != nil {
			t.Skip(err)
		}
		_, _ = terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	})
}
