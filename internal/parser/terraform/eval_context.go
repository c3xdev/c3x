package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// buildEvalContext assembles the hcl.EvalContext the parser uses to
// evaluate variable defaults, locals, attribute expressions, and meta
// arguments. The four scope objects (var, local, data, plus optional
// count/each) are passed in by the caller; everything else is the
// shared Terraform function library.
func buildEvalContext(
	vars cty.Value,
	locals cty.Value,
	data cty.Value,
	extras map[string]cty.Value,
) *hcl.EvalContext {
	variables := map[string]cty.Value{
		"var":   vars,
		"local": locals,
		"data":  data,
	}
	for k, v := range extras {
		variables[k] = v
	}
	return &hcl.EvalContext{
		Variables: variables,
		Functions: terraformFunctions(),
	}
}

// terraformFunctions registers the Terraform built-in function library.
// Most are the cty stdlib functions verbatim; type-conversion helpers
// (tostring/tonumber/etc.) come from stdlib.MakeToFunc. try() and can()
// are HCL extensions for soft-fail traversal expressions.
//
// The list intentionally mirrors Terraform's published function set so
// real-world configs don't hit "unknown function" surprises. Functions
// we deliberately don't support are documented inline.
func terraformFunctions() map[string]function.Function {
	fns := map[string]function.Function{
		// Collection helpers.
		"length":          lengthFunc,
		"concat":          stdlib.ConcatFunc,
		"contains":        stdlib.ContainsFunc,
		"distinct":        stdlib.DistinctFunc,
		"element":         stdlib.ElementFunc,
		"flatten":         stdlib.FlattenFunc,
		"keys":            stdlib.KeysFunc,
		"lookup":          stdlib.LookupFunc,
		"merge":           stdlib.MergeFunc,
		"range":           boundedRangeFunc,
		"reverse":         stdlib.ReverseListFunc,
		"setintersection": stdlib.SetIntersectionFunc,
		"setproduct":      stdlib.SetProductFunc,
		"setsubtract":     stdlib.SetSubtractFunc,
		"setunion":        stdlib.SetUnionFunc,
		"slice":           stdlib.SliceFunc,
		"sort":            stdlib.SortFunc,
		"values":          stdlib.ValuesFunc,
		"zipmap":          stdlib.ZipmapFunc,

		// Type conversions (`toset`, `tolist`, `tomap`, etc.).
		"toset":    stdlib.MakeToFunc(cty.Set(cty.DynamicPseudoType)),
		"tolist":   stdlib.MakeToFunc(cty.List(cty.DynamicPseudoType)),
		"tomap":    stdlib.MakeToFunc(cty.Map(cty.DynamicPseudoType)),
		"tostring": stdlib.MakeToFunc(cty.String),
		"tonumber": stdlib.MakeToFunc(cty.Number),
		"tobool":   stdlib.MakeToFunc(cty.Bool),

		// Strings.
		"chomp":        stdlib.ChompFunc,
		"format":       stdlib.FormatFunc,
		"formatdate":   stdlib.FormatDateFunc,
		"formatlist":   stdlib.FormatListFunc,
		"indent":       stdlib.IndentFunc,
		"join":         stdlib.JoinFunc,
		"lower":        stdlib.LowerFunc,
		"regex":        stdlib.RegexFunc,
		"regexall":     stdlib.RegexAllFunc,
		"replace":      stdlib.ReplaceFunc,
		"split":        stdlib.SplitFunc,
		"strrev":       stdlib.ReverseFunc,
		"substr":       stdlib.SubstrFunc,
		"title":        stdlib.TitleFunc,
		"trim":         stdlib.TrimFunc,
		"trimprefix":   stdlib.TrimPrefixFunc,
		"trimsuffix":   stdlib.TrimSuffixFunc,
		"trimspace":    stdlib.TrimSpaceFunc,
		"upper":        stdlib.UpperFunc,
		"jsondecode":   stdlib.JSONDecodeFunc,
		"jsonencode":   stdlib.JSONEncodeFunc,
		"csvdecode":    stdlib.CSVDecodeFunc,
		"timeadd":      stdlib.TimeAddFunc,
		"parseint":     stdlib.ParseIntFunc,
		"abs":          stdlib.AbsoluteFunc,
		"ceil":         stdlib.CeilFunc,
		"floor":        stdlib.FloorFunc,
		"log":          stdlib.LogFunc,
		"max":          stdlib.MaxFunc,
		"min":          stdlib.MinFunc,
		"pow":          stdlib.PowFunc,
		"signum":       stdlib.SignumFunc,
		"coalesce":     stdlib.CoalesceFunc,
		"coalescelist": stdlib.CoalesceListFunc,
		"compact":      stdlib.CompactFunc,
		"hasindex":     stdlib.HasIndexFunc,
		"index":        stdlib.IndexFunc,

		// HCL extensions for soft traversal.
		"try": tryfunc.TryFunc,
		"can": tryfunc.CanFunc,

		// Deliberately unsupported (require IO or live cloud calls):
		//   file(), templatefile(), filebase64(), pathexpand(),
		//   abspath(), dirname(), basename(), fileexists(),
		//   bcrypt(), uuid(), timestamp(), cidrhost(), …
		// Catalogs may not depend on these; resources that do will
		// degrade gracefully via expr-lang's AllowUndefinedVariables.
	}
	for name, fn := range extraFunctions() {
		fns[name] = fn
	}
	fns["chunklist"] = stdlib.ChunklistFunc
	return fns
}
