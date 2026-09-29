package parser_test

// Router tests for the format dispatcher. Each test sets up a
// minimal in-tree file with the right extension and content shape,
// then asserts Parse picks the right backend AND that backend
// successfully returns at least one resource.
//
// We deliberately don't mock the backends — exercising the real
// terraform / cloudformation / plan parsers proves the wire-up is
// real, not just type-correct.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/parser"
)

func TestParseRejectsEmptyPath(t *testing.T) {
	t.Parallel()
	if _, err := parser.Parse("", parser.Options{}); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestParseRejectsMissingPath(t *testing.T) {
	t.Parallel()
	if _, err := parser.Parse("/no/such/path/at/all", parser.Options{}); err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestParseRejectsUnknownExtension(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("kind = \"x\""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse(path, parser.Options{}); err == nil {
		t.Error("expected error for unsupported .toml input")
	}
}

func TestParseDirectoryDispatchesToTerraform(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "web" {
		  instance_type = "m5.xlarge"
		  ami           = "ami-x"
		}
	`)
	got, err := parser.Parse(dir, parser.Options{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_instance" {
		t.Fatalf("expected 1 aws_instance, got %+v", got)
	}
}

// A directory with CloudFormation templates and no Terraform files is
// estimated from its templates; the usage file and other YAML next to
// them are not mistaken for templates.
func TestParseDirectoryOfCloudFormationTemplates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "template.yaml"), `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  Web:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: m5.xlarge
`)
	writeFile(t, filepath.Join(dir, "storage.json"),
		`{"Resources":{"Bucket":{"Type":"AWS::S3::Bucket","Properties":{"BucketName":"x"}}}}`)
	writeFile(t, filepath.Join(dir, "c3x-usage.yml"), "version: 0.1\nresource_usage: {}\n")
	writeFile(t, filepath.Join(dir, "buildspec.yml"), "version: 0.2\nphases: {}\n")

	got, err := parser.Parse(dir, parser.Options{Region: "eu-west-1"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	kinds := map[string]bool{}
	for _, r := range got {
		kinds[r.Ref.Label()] = true
	}
	if len(got) != 2 || !kinds["aws_instance.Web"] || !kinds["aws_s3_bucket.Bucket"] {
		t.Fatalf("resources = %v, want aws_instance.Web and aws_s3_bucket.Bucket", kinds)
	}
}

// Terraform files win: a directory holding both is a Terraform
// directory, as before, and its YAML is not parsed as CloudFormation.
func TestParseDirectoryPrefersTerraformOverTemplates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
		resource "aws_instance" "web" {
		  instance_type = "m5.xlarge"
		}
	`)
	writeFile(t, filepath.Join(dir, "template.yaml"),
		"Resources:\n  Bucket:\n    Type: AWS::S3::Bucket\n")
	got, err := parser.Parse(dir, parser.Options{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_instance" {
		t.Fatalf("expected only the Terraform aws_instance, got %+v", got)
	}
}

func TestParseDirectoryWithNoConfigurationStillErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "c3x-usage.yml"), "version: 0.1\n")
	if _, err := parser.Parse(dir, parser.Options{}); err == nil {
		t.Fatal("expected an error for a directory with no configuration")
	}
}

func TestParseTFFileDispatchesToTerraform(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "single.tf")
	writeFile(t, path, `
		provider "aws" { region = "us-east-1" }
		resource "aws_s3_bucket" "b" { bucket = "x" }
	`)
	got, err := parser.Parse(path, parser.Options{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_s3_bucket" {
		t.Fatalf("expected 1 aws_s3_bucket, got %+v", got)
	}
}

func TestParseYAMLDispatchesToCloudFormation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.yaml")
	writeFile(t, path, `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
`)
	got, err := parser.Parse(path, parser.Options{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_s3_bucket" {
		t.Fatalf("expected 1 aws_s3_bucket, got %+v", got)
	}
}

func TestParseJSONSniffDispatches(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cfnPath := filepath.Join(dir, "stack.json")
	writeFile(t, cfnPath, `{"AWSTemplateFormatVersion":"2010-09-09","Resources":{"B":{"Type":"AWS::S3::Bucket","Properties":{"BucketName":"x"}}}}`)
	got, err := parser.Parse(cfnPath, parser.Options{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("CFN JSON: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_s3_bucket" {
		t.Errorf("CFN JSON: expected aws_s3_bucket, got %+v", got)
	}

	planPath := filepath.Join(dir, "plan.json")
	writeFile(t, planPath, `{
	  "format_version": "1.2",
	  "resource_changes": [{
	    "address": "aws_instance.web",
	    "type": "aws_instance",
	    "name": "web",
	    "change": {
	      "actions": ["create"],
	      "after": {"instance_type": "t3.micro", "ami": "ami-x"}
	    }
	  }]
	}`)
	got, err = parser.Parse(planPath, parser.Options{})
	if err != nil {
		t.Fatalf("plan JSON: %v", err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_instance" {
		t.Errorf("plan JSON: expected aws_instance, got %+v", got)
	}
}

func TestParseExplicitCFNExtension(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.cfn.yaml")
	writeFile(t, path, `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-bucket
`)
	got, err := parser.Parse(path, parser.Options{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 resource, got %d", len(got))
	}
}

func TestParseUnsupportedExtensionErrorMessage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "weird.bicep")
	writeFile(t, path, "anything")
	_, err := parser.Parse(path, parser.Options{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// A .tf.json path is configuration, not a plan: it goes to the HCL
// parser, and IsPlanFile says no.
func TestParseTFJSONFileIsConfiguration(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "main.tf.json")
	if err := os.WriteFile(p, []byte(`{"resource":{"aws_instance":{"web":{"instance_type":"m5.large"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if parser.IsPlanFile(p) {
		t.Error("a .tf.json file must not be treated as a plan")
	}
	got, err := parser.Parse(p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Ref.Kind != "aws_instance" || got[0].Attributes["instance_type"] != "m5.large" {
		t.Fatalf("got %+v, want one aws_instance with instance_type m5.large", got)
	}
}
