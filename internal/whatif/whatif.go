// Package whatif applies `<resource address>.<attr>=value` CLI overrides to
// parsed resources. The overrides land between parser and calculator
// — after the IaC source is resolved into [domain.Resource]s but
// before the calculator evaluates dimensions — so users can ask
// "what if this aws_instance was m6i.large?" without editing the .tf.
//
// Overrides participate in the same precedence chain as everything
// else: parser defaults < usage file < --what-if. The whatif layer
// is applied last so it always wins.
package whatif

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
)

// Override is one parsed override directive. It binds an attribute on
// a specific resource to a typed value. The resource is written as its
// Terraform address (`module.web.aws_instance.this[0].instance_type`).
type Override struct {
	Kind  string
	Name  string
	Attr  string
	Value any
	// Legacy is set when the address used the pre-0.3.19 form, with the
	// kind in front of the module path (aws_instance.module.web.this).
	// It still matches; the caller can suggest the new form.
	Legacy bool
}

// Address is the override's resource in Terraform address form.
func (o Override) Address() string {
	return domain.Reference{Kind: o.Kind, Name: o.Name}.TerraformAddress()
}

// Parse turns a slice of `--what-if <address>.<attr>=value` strings
// into [Override]s. Values are type-coerced: bool first, then int,
// then float, then string. The strict type-precedence order keeps
// `true` from accidentally becoming the string "true".
func Parse(raws []string) ([]Override, error) {
	out := make([]Override, 0, len(raws))
	for _, raw := range raws {
		o, err := parseOne(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func parseOne(raw string) (Override, error) {
	eq := strings.Index(raw, "=")
	if eq < 0 {
		return Override{}, fmt.Errorf("--what-if expects <resource address>.<attr>=value, got %q", raw)
	}
	lhs, rhs := raw[:eq], raw[eq+1:]
	parts := splitAddress(lhs)
	if len(parts) < 3 {
		return Override{}, fmt.Errorf("--what-if %q: need <resource address>.<attr>, e.g. aws_instance.web.instance_type", lhs)
	}
	attr := parts[len(parts)-1]
	addr := parts[:len(parts)-1]

	// Terraform address: module.<name>[.module.<name>...].<kind>.<name>
	if addr[0] == "module" {
		i := 0
		var modules []string
		for i+1 < len(addr) && addr[i] == "module" {
			modules = append(modules, addr[i], addr[i+1])
			i += 2
		}
		if len(addr)-i != 2 {
			return Override{}, fmt.Errorf("--what-if %q: expected module.<name>...<kind>.<name>.<attr>", lhs)
		}
		name := strings.Join(append(modules, addr[i+1]), ".")
		return Override{Kind: addr[i], Name: name, Attr: attr, Value: coerce(rhs)}, nil
	}

	// <kind>.<name>, or the pre-0.3.19 <kind>.module.<name>...<name>.
	return Override{
		Kind:   addr[0],
		Name:   strings.Join(addr[1:], "."),
		Attr:   attr,
		Value:  coerce(rhs),
		Legacy: len(addr) > 2 && addr[1] == "module",
	}, nil
}

// splitAddress splits on dots outside [...], so an index or for_each key
// that contains a dot (web["a.b"]) stays one segment.
func splitAddress(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// coerce narrows a raw value string into the strongest Go type that
// fits. Booleans win first so `true` doesn't accidentally become a
// string; the catalog's `pick(cond, a, b)` would silently break if
// `default(monitored, false)` started returning a string.
func coerce(v string) any {
	switch v {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	// Strip surrounding quotes so `--what-if 'x.y.z="abc"'` doesn't
	// produce the string `"abc"` with literal quotes.
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[0] == v[len(v)-1] {
		return v[1 : len(v)-1]
	}
	return v
}

// Apply mutates `resources` in place, layering every Override onto
// the matching resource. Returns the list of overrides that didn't
// match any resource so the caller can surface a warning rather than
// silently dropping them.
func Apply(resources []domain.Resource, overrides []Override) (unmatched []Override) {
	for _, o := range overrides {
		matched := false
		for i, r := range resources {
			if r.Ref.Kind == o.Kind && r.Ref.Name == o.Name {
				if resources[i].Attributes == nil {
					resources[i].Attributes = map[string]any{}
				}
				resources[i].Attributes[o.Attr] = o.Value
				matched = true
			}
		}
		if !matched {
			unmatched = append(unmatched, o)
		}
	}
	return unmatched
}
