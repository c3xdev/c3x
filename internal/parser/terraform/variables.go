package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// collectVariableDefaults extracts each `variable "x" { default = ... }`
// default value, keyed by variable name. The returned map is the
// starting point for the var scope; later stages layer tfvars and CLI
// overrides on top.
func collectVariableDefaults(sources []sourceFile) (map[string]cty.Value, error) {
	out := map[string]cty.Value{}
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "variable" || len(block.Labels) == 0 {
				continue
			}
			name := block.Labels[0]
			defaultAttr, ok := block.Body.Attributes["default"]
			if !ok {
				continue
			}
			ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
			val, diags := defaultAttr.Expr.Value(ctx)
			if diags.HasErrors() {
				return nil, fmt.Errorf("variable %q default in %s: %s",
					name, src.Path, formatDiags(diags))
			}
			out[name] = val
		}
	}
	return out, nil
}

// variableTypeConstraint is a variable's declared `type` constraint plus
// the optional-attribute defaults derived from it. Applying it to a value
// mirrors what Terraform does when it decodes a variable: object attributes
// declared `optional(...)` that are absent from the given value are filled
// with their declared default (or null), and the value is then converted to
// the constraint type.
//
// This matters for registry modules like terraform-aws-modules/rds-aurora,
// whose `instances` input is typed `map(object({ instance_class =
// optional(string), ... }))`. Without the constraint, a caller value of
// `{ one = {} }` stays an empty object, so a downstream traversal such as
// `try(coalesce(each.value.instance_class, var.cluster_instance_class), null)`
// errors on the missing attribute, `try` swallows the whole expression to
// null, and the resource silently falls back to the catalogue default class.
// With the constraint applied, `each.value.instance_class` resolves to null
// and the coalesce correctly picks up the module variable.
type variableTypeConstraint struct {
	ty       cty.Type
	defaults *typeexpr.Defaults
}

// apply fills optional-attribute defaults and converts val to the declared
// type. It is best-effort: a value that can't be converted (an unexpected
// shape) is returned unchanged so estimation degrades gracefully rather than
// dropping the resource. Null and unknown values are left untouched.
func (c variableTypeConstraint) apply(val cty.Value) cty.Value {
	if val.IsNull() || !val.IsKnown() {
		return val
	}
	out := val
	if c.defaults != nil {
		out = c.defaults.Apply(out)
	}
	converted, err := convert.Convert(out, c.ty)
	if err != nil {
		return val
	}
	return converted
}

// collectVariableTypes reads each `variable "x" { type = ... }` constraint,
// keyed by variable name. Variables without a `type`, or typed `any`
// (cty.DynamicPseudoType), are omitted: `any` imposes no constraint, and
// forcing a conversion for it would strip useful structure. Unparseable
// type expressions are skipped rather than failing the estimate.
func collectVariableTypes(sources []sourceFile) map[string]variableTypeConstraint {
	out := map[string]variableTypeConstraint{}
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "variable" || len(block.Labels) == 0 {
				continue
			}
			typeAttr, ok := block.Body.Attributes["type"]
			if !ok {
				continue
			}
			ty, defaults, diags := typeexpr.TypeConstraintWithDefaults(typeAttr.Expr)
			if diags.HasErrors() || ty == cty.DynamicPseudoType {
				continue
			}
			out[block.Labels[0]] = variableTypeConstraint{ty: ty, defaults: defaults}
		}
	}
	return out
}

// applyVariableTypes converts every value in vars that has a declared type
// constraint, in place. Call it once the var scope is fully layered
// (defaults + tfvars + CLI, or defaults + tfvars + module inputs) so the
// final value — whatever its source — is normalised the way Terraform would
// normalise it before locals, meta-arguments, and attributes read it.
func applyVariableTypes(vars map[string]cty.Value, types map[string]variableTypeConstraint) {
	for name, c := range types {
		if v, ok := vars[name]; ok {
			vars[name] = c.apply(v)
		}
	}
}

// applyAutoTfvars layers Terraform's auto-loading tfvars files onto the
// variables map, in the order Terraform itself loads them:
//
//	terraform.tfvars
//	terraform.tfvars.json
//	*.auto.tfvars       (lexical)
//	*.auto.tfvars.json  (lexical)
//
// A missing optional file is not an error; a present-but-malformed file
// is.
func applyAutoTfvars(dir string, vars map[string]cty.Value) error {
	for _, name := range []string{"terraform.tfvars", "terraform.tfvars.json"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			if err := applyVarFile(p, vars); err != nil {
				return err
			}
		}
	}
	for _, pattern := range []string{"*.auto.tfvars", "*.auto.tfvars.json"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		sort.Strings(matches)
		for _, p := range matches {
			if err := applyVarFile(p, vars); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyVarFile reads a tfvars file (HCL or JSON depending on extension)
// and merges its assignments into the variables map. Later sources
// override earlier ones — caller controls the order.
func applyVarFile(path string, vars map[string]cty.Value) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return &errInvalidVarFile{path: path, err: err}
	}
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return &errInvalidVarFile{path: path, err: err}
		}
		for k, v := range decoded {
			vars[k] = anyToCty(v)
		}
		return nil
	}
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(raw, path)
	if diags.HasErrors() {
		return &errInvalidVarFile{path: path, err: fmt.Errorf("%s", formatDiags(diags))}
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return &errInvalidVarFile{path: path, err: fmt.Errorf("unexpected body type %T", file.Body)}
	}
	ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
	for _, attr := range body.Attributes {
		val, vdiags := attr.Expr.Value(ctx)
		if vdiags.HasErrors() {
			return &errInvalidVarFile{
				path: path,
				err:  fmt.Errorf("attribute %q: %s", attr.Name, formatDiags(vdiags)),
			}
		}
		vars[attr.Name] = val
	}
	return nil
}

// parseCLIVar converts a `--var name=value` raw string into a cty.Value.
// Terraform supports either a bare string or an HCL expression on the
// right-hand side; we try HCL first and fall back to a string literal.
func parseCLIVar(raw string) cty.Value {
	parser := hclparse.NewParser()
	wrapper := []byte(fmt.Sprintf("x = %s", raw))
	file, diags := parser.ParseHCL(wrapper, "<cli>")
	if diags.HasErrors() {
		return cty.StringVal(raw)
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return cty.StringVal(raw)
	}
	for _, attr := range body.Attributes {
		ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
		val, vdiags := attr.Expr.Value(ctx)
		if vdiags.HasErrors() {
			return cty.StringVal(raw)
		}
		return val
	}
	return cty.StringVal(raw)
}

// asObject wraps a flat variables map into a single cty.Object value
// the evaluator addresses as `var.x` / `local.x`. The wrapper exists
// because hcl.EvalContext.Variables expects top-level scopes, and HCL
// looks up traversals like `var.region` by reading `var` from the map
// then projecting `.region` off the resulting object.
func asObject(m map[string]cty.Value) cty.Value {
	if len(m) == 0 {
		return cty.EmptyObjectVal
	}
	return cty.ObjectVal(m)
}
