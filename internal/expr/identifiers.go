package expr

import (
	"sort"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// Identifiers returns the names an expression reads from its environment
// — resource attributes and usage keys — excluding function calls and
// the functions and constants the evaluator itself provides. The
// calculator uses it to tell which attributes a price depends on, so an
// unresolved attribute or a missing usage value is reported only where
// it actually changes the number.
func Identifiers(src string) ([]string, error) {
	if src == "" {
		return nil, nil
	}
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	v := &identVisitor{seen: map[string]bool{}, called: map[string]bool{}}
	ast.Walk(&tree.Node, v)
	provided := EnvFor(domain.Resource{}, nil, nil)
	var out []string
	for name := range v.seen {
		if v.called[name] {
			continue
		}
		if _, builtin := provided[name]; builtin {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

type identVisitor struct {
	seen   map[string]bool
	called map[string]bool
}

func (v *identVisitor) Visit(node *ast.Node) { //nolint:gocritic // signature fixed by ast.Visitor
	switch n := (*node).(type) {
	case *ast.IdentifierNode:
		v.seen[n.Value] = true
	case *ast.CallNode:
		if id, ok := n.Callee.(*ast.IdentifierNode); ok {
			v.called[id.Value] = true
		}
	}
}
