package expr

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Program is a compiled expression. Compilation is cached by callers so
// the same expression text isn't re-parsed for every resource. The zero
// value is invalid; construct via Compile.
type Program struct {
	source string
	prog   *vm.Program
}

// Source returns the original expression text for diagnostics.
func (p Program) Source() string { return p.source }

// Compile parses and compiles an expression once. Subsequent calls to
// Run share the parse cost. expr.AllowUndefinedVariables lets the
// catalog reference attributes that may or may not be present in any
// given resource — they evaluate to their zero value and the catalog's
// `default(x, fallback)` idiom handles substitution.
func Compile(source string) (Program, error) {
	if source == "" {
		return Program{}, fmt.Errorf("empty expression")
	}
	p, err := expr.Compile(source, expr.AllowUndefinedVariables())
	if err != nil {
		return Program{}, fmt.Errorf("compiling %q: %w", source, err)
	}
	return Program{source: source, prog: p}, nil
}

// MustCompile panics if compilation fails. Used in package init and
// tests where a failure to compile is a programming error.
func MustCompile(source string) Program {
	p, err := Compile(source)
	if err != nil {
		panic(err)
	}
	return p
}

// Run executes the compiled program against the given environment and
// returns the result. Use the type-narrowing helpers below to avoid
// callers re-doing the cast.
func Run(p Program, env map[string]any) (any, error) {
	if p.prog == nil {
		return nil, fmt.Errorf("running uninitialised program")
	}
	out, err := expr.Run(p.prog, env)
	if err != nil {
		return nil, fmt.Errorf("evaluating %q: %w", p.source, err)
	}
	return out, nil
}

// RunBool runs p and returns the result as a bool. An untyped nil is
// treated as false (the catalog `when` idiom: omitted predicate → row
// is always emitted). Numbers, strings, and arrays cause an error
// because that signals a catalog authoring mistake.
func RunBool(p Program, env map[string]any) (bool, error) {
	v, err := Run(p, env)
	if err != nil {
		return false, err
	}
	switch t := v.(type) {
	case nil:
		return false, nil
	case bool:
		return t, nil
	}
	return false, fmt.Errorf("expression %q: expected bool, got %T", p.source, v)
}

// RunString runs p and returns the result as a string. Numeric values
// are stringified via fmt.Sprint for the (common) case where an
// attribute filter expression like `default(instance_type, "t3.micro")`
// could resolve to a number via the env's coercion.
func RunString(p Program, env map[string]any) (string, error) {
	v, err := Run(p, env)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return fmt.Sprint(v), nil
}

// RunNumber runs p and returns the result as a float64. Booleans are
// projected to 0/1; nil is 0. Strings that parse as numbers also work,
// because the catalog occasionally stores numeric-looking values as
// strings in TOML and the expression layer should be tolerant.
func RunNumber(p Program, env map[string]any) (float64, error) {
	v, err := Run(p, env)
	if err != nil {
		return 0, err
	}
	switch t := v.(type) {
	case nil:
		return 0, nil
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case string:
		// CloudFormation accepts strings for numeric properties
		// (`AllocatedStorage: "250"`), and a Terraform variable typed
		// string can carry a number the same way.
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, nil
		}
	}
	return 0, fmt.Errorf("expression %q: expected number, got %T", p.source, v)
}
