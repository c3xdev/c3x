package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// Expected values are the documented examples from the Terraform and
// OpenTofu function references, so these check semantics, not just that
// the functions exist.
func TestExtraFunctions(t *testing.T) {
	cases := []struct{ expr, want string }{
		{`cidrsubnet("172.16.0.0/12", 4, 2)`, `"172.18.0.0/16"`},
		{`cidrsubnet("10.1.2.0/24", 4, 15)`, `"10.1.2.240/28"`},
		{`cidrsubnet("fd00:fd12:3456:7890::/56", 16, 162)`, `"fd00:fd12:3456:7800:a200::/72"`},
		{`cidrhost("10.12.112.0/20", 16)`, `"10.12.112.16"`},
		{`cidrhost("10.12.112.0/20", 268)`, `"10.12.113.12"`},
		{`cidrhost("fd00:fd12:3456:7890:00a2::/72", 34)`, `"fd00:fd12:3456:7890::22"`},
		{`cidrhost("10.0.0.0/24", -1)`, `"10.0.0.255"`},
		{`cidrnetmask("172.16.0.0/12")`, `"255.240.0.0"`},
		{`cidrsubnets("10.1.0.0/16", 4, 4, 8, 4)`, `["10.1.0.0/20","10.1.16.0/20","10.1.32.0/24","10.1.48.0/20"]`},
		{`cidrsubnets("fd00:fd12:3456:7890::/56", 16, 16, 16, 32)`, `["fd00:fd12:3456:7800::/72","fd00:fd12:3456:7800:100::/72","fd00:fd12:3456:7800:200::/72","fd00:fd12:3456:7800:300::/88"]`},
		{`cidrcontains("192.168.2.0/20", "192.168.2.1")`, `true`},
		{`cidrcontains("192.168.2.0/20", "192.126.2.1")`, `false`},
		{`cidrcontains("10.0.0.0/8", "10.1.0.0/16")`, `true`},
		{`one([])`, `null`},
		{`one(["hello"])`, `"hello"`},
		{`sum([10, 13, 6, 4.5])`, `33.5`},
		{`alltrue(["true", true])`, `true`},
		{`alltrue([true, false])`, `false`},
		{`anytrue([false, true])`, `true`},
		{`anytrue([])`, `false`},
		{`startswith("hello world", "hello")`, `true`},
		{`endswith("hello world", "world")`, `true`},
		{`strcontains("hello world", "wor")`, `true`},
		{`templatestring("Hello, $${name}!", { name = "Jodie" })`, `"Hello, Jodie!"`},
		{`templatestring("m5.$${size}", { size = "large" })`, `"m5.large"`},
		{`nonsensitive("m5.large")`, `"m5.large"`},
		{`issensitive("x")`, `false`},
		{`base64encode("Hello World")`, `"SGVsbG8gV29ybGQ="`},
		{`base64decode("SGVsbG8gV29ybGQ=")`, `"Hello World"`},
		{`base64gunzip(base64gzip("round trip"))`, `"round trip"`},
		{`urlencode("Hello World!")`, `"Hello+World%21"`},
		{`urldecode("Hello+World%21")`, `"Hello World!"`},
		{`md5("hello world")`, `"5eb63bbbe01eeed093cb22bb8f5acdc3"`},
		{`sha1("hello world")`, `"2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"`},
		{`sha256("hello world")`, `"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"`},
		{`base64sha256("hello world")`, `"uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek="`},
		{`yamldecode("hello: world")`, `{"hello":"world"}`},
		{`yamldecode("instances:\n  - m5.large\n  - m5.xlarge")["instances"][1]`, `"m5.xlarge"`},
		{`chunklist(["a", "b", "c"], 2)`, `[["a","b"],["c"]]`},
	}
	ctx := &hcl.EvalContext{Functions: terraformFunctions()}
	for _, c := range cases {
		expr, diags := hclsyntax.ParseExpression([]byte(c.expr), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Errorf("%s: parse: %s", c.expr, diags)
			continue
		}
		v, diags := expr.Value(ctx)
		if diags.HasErrors() {
			t.Errorf("%s: %s", c.expr, diags)
			continue
		}
		if got := ctyToJSONString(t, v); got != c.want {
			t.Errorf("%s = %s, want %s", c.expr, got, c.want)
		}
	}
}

func ctyToJSONString(t *testing.T, v cty.Value) string {
	t.Helper()
	if v.IsNull() {
		return "null"
	}
	b, err := ctyjson.Marshal(v, v.Type())
	if err != nil {
		t.Fatalf("marshal %#v: %v", v, err)
	}
	return string(b)
}
