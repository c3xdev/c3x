package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestApplyOptionalDefaults_FillsMissingKeys(t *testing.T) {
	t.Parallel()

	src := `
variable "vm" {
  type = object({
    size     = optional(string, "Standard_D2s_v3")
    os_disk  = optional(object({
      disk_size_gb         = optional(number, 64)
      storage_account_type = optional(string, "Premium_LRS")
    }), {})
  })
}
`
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(src), "test.tf")
	if diags.HasErrors() {
		t.Fatalf("parse: %s", diags.Error())
	}
	body := file.Body.(*hclsyntax.Body)
	sources := []sourceFile{{Path: "test.tf", Body: body}}

	// Caller supplies only { dataDisks = [...] } — no size, no os_disk.
	vars := map[string]cty.Value{
		"vm": cty.ObjectVal(map[string]cty.Value{
			"dataDisks": cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	}

	applyOptionalDefaults(sources, vars)

	result := vars["vm"]
	if !result.Type().HasAttribute("size") {
		t.Fatal("expected 'size' attribute to be filled in")
	}
	size := result.GetAttr("size")
	if size.AsString() != "Standard_D2s_v3" {
		t.Errorf("size = %q, want %q", size.AsString(), "Standard_D2s_v3")
	}

	if !result.Type().HasAttribute("os_disk") {
		t.Fatal("expected 'os_disk' attribute to be filled in")
	}
	osDisk := result.GetAttr("os_disk")
	if !osDisk.IsKnown() {
		t.Error("os_disk should be known")
	}
}

func TestApplyOptionalDefaults_PreservesExistingKeys(t *testing.T) {
	t.Parallel()

	src := `
variable "config" {
  type = object({
    name   = optional(string, "default-name")
    region = optional(string, "us-east-1")
  })
}
`
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(src), "test.tf")
	if diags.HasErrors() {
		t.Fatalf("parse: %s", diags.Error())
	}
	body := file.Body.(*hclsyntax.Body)
	sources := []sourceFile{{Path: "test.tf", Body: body}}

	// Caller supplies name but not region.
	vars := map[string]cty.Value{
		"config": cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal("my-custom-name"),
		}),
	}

	applyOptionalDefaults(sources, vars)

	result := vars["config"]
	name := result.GetAttr("name")
	if name.AsString() != "my-custom-name" {
		t.Errorf("name should be preserved: got %q", name.AsString())
	}
	region := result.GetAttr("region")
	if region.AsString() != "us-east-1" {
		t.Errorf("region should be filled from default: got %q", region.AsString())
	}
}

func TestExtractOptionalDefaults_NonObjectType(t *testing.T) {
	t.Parallel()

	// variable with type = string — no optional() to extract
	src := `variable "simple" { type = string }`
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(src), "test.tf")
	if diags.HasErrors() {
		t.Fatalf("parse: %s", diags.Error())
	}
	body := file.Body.(*hclsyntax.Body)
	sources := []sourceFile{{Path: "test.tf", Body: body}}

	vars := map[string]cty.Value{
		"simple": cty.StringVal("hello"),
	}

	// Should not panic or modify the value.
	applyOptionalDefaults(sources, vars)

	if vars["simple"].AsString() != "hello" {
		t.Errorf("unexpected modification: %v", vars["simple"])
	}
}
