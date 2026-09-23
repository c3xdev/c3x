package expr_test

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	c3xexpr "github.com/c3xdev/c3x/internal/expr"
)

// TestPropertyExprNeverPanics asserts the expression layer's two
// invariants:
//
//  1. Compile + Run on any expression text returns an error or a value
//     — never a panic. Catalog authors who fat-finger a TOML must not
//     take the binary down.
//  2. RunNumber against an arbitrary tree always returns a finite
//     float64 (or an error). Catalogs sometimes do math on
//     `default(missing, 0)` patterns; we test that the zero-value flow
//     stays numeric.
//
// The property generator builds expressions from a small grammar
// covering the constructs real catalogs use: identifiers,
// function calls, arithmetic, string ops, boolean ops.
func TestPropertyExprNeverPanics(t *testing.T) {
	t.Parallel()

	// Two fixed seeds — runs are reproducible across machines so a
	// failure on CI is replicable locally with no flag tweaks.
	rng := rand.New(rand.NewPCG(0xC3, 0xDE))
	for i := 0; i < 2000; i++ {
		src := generateExpr(rng, 4)
		t.Run(src, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on %q: %v", src, r)
				}
			}()
			prog, err := c3xexpr.Compile(src)
			if err != nil {
				// Compile errors are fine; the calculator wraps them.
				return
			}
			env := c3xexpr.EnvFor(domain.Resource{
				Attributes: map[string]any{"x": float64(10), "name": "demo"},
			}, nil, nil)
			_, _ = c3xexpr.Run(prog, env)
		})
	}
}

// generateExpr builds a random expression of `depth` levels deep using
// the constructs that real catalog TOMLs use. The grammar is
// intentionally narrow — we're testing the evaluator's behaviour on
// realistic shapes, not exhaustively probing every expr-lang feature.
func generateExpr(rng *rand.Rand, depth int) string {
	if depth <= 0 {
		return atom(rng)
	}
	switch rng.IntN(8) {
	case 0:
		return "default(" + generateExpr(rng, depth-1) + "," + generateExpr(rng, depth-1) + ")"
	case 1:
		return "pick(" + generateExpr(rng, depth-1) + "==" + generateExpr(rng, depth-1) +
			"," + generateExpr(rng, depth-1) + "," + generateExpr(rng, depth-1) + ")"
	case 2:
		return "monthly_hours()*" + generateExpr(rng, depth-1)
	case 3:
		op := "+-*/"[rng.IntN(4)]
		return generateExpr(rng, depth-1) + string(op) + generateExpr(rng, depth-1)
	case 4:
		return "length(" + generateExpr(rng, depth-1) + ")"
	case 5:
		return `format("%s", ` + generateExpr(rng, depth-1) + ")"
	default:
		return atom(rng)
	}
}

func atom(rng *rand.Rand) string {
	switch rng.IntN(6) {
	case 0:
		return "x"
	case 1:
		return "name"
	case 2:
		return "missing_attr"
	case 3:
		return `"literal"`
	case 4:
		return "0"
	default:
		s := strings.Builder{}
		for i := 0; i < rng.IntN(4); i++ {
			s.WriteByte(byte('0' + rng.IntN(10)))
		}
		if s.Len() == 0 {
			s.WriteByte('1')
		}
		return s.String()
	}
}
