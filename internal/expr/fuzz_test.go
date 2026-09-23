package expr_test

// Fuzz tests for the expression layer. The goal is to surface
// crashes on malformed input — every error path must return an
// error, never panic. expr-lang's evaluator is reasonably robust on
// its own but our wrapper (Compile / RunString / RunNumber /
// RunBool) adds type-coercion logic that can mis-handle weird
// inputs without an explicit guard.
//
// Run with `go test ./internal/expr/... -fuzz=FuzzCompile -fuzztime=10s`.

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	c3xexpr "github.com/c3xdev/c3x/internal/expr"
)

func FuzzCompile(f *testing.F) {
	// Seed the corpus with a representative slice of the catalog's
	// expression dialect so the mutator starts from realistic shapes.
	seeds := []string{
		`monthly_hours()`,
		`default(size, 100)`,
		`default(engine, "postgres") == "mysql"`,
		`price("compute")`,
		`pick(default(multi_az, false), price("multi"), price("single"))`,
		`default(monthly_io_requests, 0) / 1000000`,
		`default(size, 100) * 0.08`,
		`replace(default(host_instance_type, "mq.t3.micro"), "mq.", "")`,
		`string(default(vcores, 2)) + " vCore"`,
		``,
		`)`,
		`((((`,
		`"`,
		`default(`,
		`/0`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		// Compilation must never panic. An error is acceptable.
		prog, err := c3xexpr.Compile(src)
		if err != nil {
			return
		}
		// If compilation succeeded, evaluation against an empty env
		// must also not panic. Type errors / nil deref → return an
		// error from RunString/Run.
		env := c3xexpr.EnvFor(domain.Resource{}, nil, nil)
		// Best-effort run each accessor. We don't care about the
		// values, only that the call returns rather than panics.
		_, _ = c3xexpr.Run(prog, env)
		_, _ = c3xexpr.RunString(prog, env)
		_, _ = c3xexpr.RunNumber(prog, env)
		_, _ = c3xexpr.RunBool(prog, env)
	})
}
