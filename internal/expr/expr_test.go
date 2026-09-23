package expr_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	c3xexpr "github.com/c3xdev/c3x/internal/expr"
	"github.com/shopspring/decimal"
)

func mustCompile(t *testing.T, src string) c3xexpr.Program {
	t.Helper()
	p, err := c3xexpr.Compile(src)
	if err != nil {
		t.Fatalf("Compile(%q): %v", src, err)
	}
	return p
}

func TestStdlibPrice(t *testing.T) {
	t.Parallel()

	lookup := func(name string) (decimal.Decimal, string, error) {
		if name == "compute" {
			return decimal.RequireFromString("0.192"), domain.PriceSourceLive, nil
		}
		return decimal.Zero, domain.PriceSourceLive, nil
	}
	env := c3xexpr.EnvFor(domain.Resource{}, lookup, nil)
	p := mustCompile(t, `price("compute")`)
	got, err := c3xexpr.RunNumber(p, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0.192 {
		t.Errorf("price(\"compute\") = %v, want 0.192", got)
	}
}

func TestStdlibDefault(t *testing.T) {
	t.Parallel()

	r := domain.Resource{Attributes: map[string]any{
		"instance_type": "m5.xlarge",
		"empty_string":  "",
	}}
	env := c3xexpr.EnvFor(r, nil, nil)

	cases := []struct {
		src  string
		want string
	}{
		{`default(instance_type, "t3.micro")`, "m5.xlarge"},
		{`default(missing_attr, "t3.micro")`, "t3.micro"},
		{`default(empty_string, "fallback")`, "fallback"},
	}
	for _, tc := range cases {
		p := mustCompile(t, tc.src)
		got, err := c3xexpr.RunString(p, env)
		if err != nil {
			t.Fatalf("%q: %v", tc.src, err)
		}
		if got != tc.want {
			t.Errorf("%q = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestStdlibMonthlyHours(t *testing.T) {
	t.Parallel()

	env := c3xexpr.EnvFor(domain.Resource{}, nil, nil)
	p := mustCompile(t, `monthly_hours()`)
	got, err := c3xexpr.RunNumber(p, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != 730 {
		t.Errorf("monthly_hours() = %v, want 730", got)
	}
}

func TestStdlibPickFunction(t *testing.T) {
	t.Parallel()

	env := c3xexpr.EnvFor(domain.Resource{Attributes: map[string]any{
		"size": "large",
	}}, nil, nil)

	p := mustCompile(t, `pick(size == "large", 100, 10)`)
	got, err := c3xexpr.RunNumber(p, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Errorf("pick(...) = %v, want 100", got)
	}
}

func TestUndefinedVariablesYieldZeroValues(t *testing.T) {
	t.Parallel()

	// `vcpu_count` isn't set on this resource. Without
	// AllowUndefinedVariables this would be a compile error; with it,
	// the catalog's `default(vcpu_count, 2)` idiom resolves cleanly.
	env := c3xexpr.EnvFor(domain.Resource{}, nil, nil)
	p := mustCompile(t, `default(vcpu_count, 2) * monthly_hours()`)
	got, err := c3xexpr.RunNumber(p, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2*730 {
		t.Errorf("expected 1460, got %v", got)
	}
}

func TestRunStringStringifiesNumerics(t *testing.T) {
	t.Parallel()

	env := c3xexpr.EnvFor(domain.Resource{}, nil, nil)
	p := mustCompile(t, `42`)
	got, err := c3xexpr.RunString(p, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("RunString on 42 = %q, want %q", got, "42")
	}
}

func TestRunBoolRejectsNonBool(t *testing.T) {
	t.Parallel()

	env := c3xexpr.EnvFor(domain.Resource{}, nil, nil)
	p := mustCompile(t, `"not a bool"`)
	if _, err := c3xexpr.RunBool(p, env); err == nil {
		t.Errorf("expected error when expression doesn't return bool")
	}
}

func TestCompileEmptyFails(t *testing.T) {
	t.Parallel()

	if _, err := c3xexpr.Compile(""); err == nil {
		t.Errorf("expected error on empty expression")
	}
}
