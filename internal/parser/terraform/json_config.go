package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// jsonToNative translates a Terraform JSON configuration file (.tf.json,
// .tofu.json) into equivalent native-syntax source, which loadFiles then
// parses like any .tf file.
//
// Why translate rather than use HCL's JSON parser directly: every stage of
// this package walks *hclsyntax.Body (blocks, attributes, expressions),
// and the JSON body HCL produces is schema-driven, so it can't be walked
// that way. Terraform decodes it with provider schemas that say which
// keys are attributes and which are nested blocks; c3x has no provider
// schemas. Translating keeps one AST type throughout and needs no schema:
//
//   - The top-level structure is fixed by the language (resource, data,
//     module, provider, variable, locals), so it becomes native blocks.
//   - Inside a resource, a JSON object or array of objects is written as
//     an attribute: `root_block_device = { volume_size = 50 }`. That
//     evaluates to exactly the attribute shape a literal nested block
//     produces (a map, or a list of maps for a repeated block), which is
//     all the calculator reads.
//   - Strings are templates in Terraform JSON, as quoted strings are in
//     native syntax, so "${var.size}" carries over unchanged, and a
//     string that is a single interpolation yields the raw value, as it
//     does in native syntax: "count": "${length(var.azs)}" is a number.
//   - `dynamic` becomes a native dynamic block, so it expands the same way.
//   - References Terraform JSON writes as bare strings are written as
//     references: a resource's `provider`, a module's `providers` map,
//     a variable's `type`, and a dynamic block's `iterator`.
//
// Skipped, as irrelevant to cost: output, terraform, moved, import, check
// and removed blocks, and depends_on, lifecycle, provisioner and
// connection inside resources. Anything else this can't represent is
// skipped with a warning naming the file and key, never silently.
//
// Diagnostics for a JSON file cite positions in the translated source,
// not the original JSON; the file name is the original.
func jsonToNative(raw []byte, path string, logger *slog.Logger) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	root, err := decodeOrdered(dec, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid JSON: unexpected data after the top-level object")
	}
	obj, ok := root.(*jsonObject)
	if !ok {
		return nil, errors.New("the top level of a JSON configuration file must be an object")
	}
	t := &jsonTranslator{path: path, logger: logger}
	t.file(obj)
	return []byte(t.b.String()), nil
}

// jsonObject is a decoded JSON object with its key order and duplicate
// keys preserved: Terraform JSON repeats keys (two "resource" objects)
// and block order matters.
type jsonObject struct {
	keys []string
	vals []any
}

// maxJSONDepth bounds nesting so a crafted file can't exhaust the stack.
const maxJSONDepth = 256

func decodeOrdered(dec *json.Decoder, depth int) (any, error) {
	if depth > maxJSONDepth {
		return nil, fmt.Errorf("nested deeper than %d levels", maxJSONDepth)
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := &jsonObject{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("object key %v is not a string", kt)
				}
				v, err := decodeOrdered(dec, depth+1)
				if err != nil {
					return nil, err
				}
				obj.keys = append(obj.keys, key)
				obj.vals = append(obj.vals, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return obj, nil
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := decodeOrdered(dec, depth+1)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("unexpected %q", t)
	default:
		return t, nil
	}
}

type jsonTranslator struct {
	b      strings.Builder
	path   string
	logger *slog.Logger
}

// bodyKind selects which keys of a block body get special treatment.
type bodyKind int

const (
	bodyResource bodyKind = iota // resource, data, and dynamic content
	bodyModule
	bodyVariable
	bodyPlain // provider, locals: no special keys
)

func (t *jsonTranslator) skip(reason, key string) {
	t.logger.Warn("unsupported construct in JSON configuration skipped; "+
		"anything it declares is missing from the estimate",
		"file", t.path, "key", key, "reason", reason)
}

func (t *jsonTranslator) file(obj *jsonObject) {
	for i, key := range obj.keys {
		val := obj.vals[i]
		switch key {
		case "//", "output", "terraform", "moved", "import", "check", "removed":
			// Comments, and blocks that declare nothing c3x prices.
		case "resource", "data":
			t.typedBlocks(key, val)
		case "module":
			t.namedBlocks(key, val, bodyModule)
		case "provider":
			t.namedBlocks(key, val, bodyPlain)
		case "variable":
			t.namedBlocks(key, val, bodyVariable)
		case "locals":
			t.forEachObject(key, val, func(o *jsonObject) {
				t.b.WriteString("locals {\n")
				t.attributes(o, bodyPlain, "locals")
				t.b.WriteString("}\n")
			})
		default:
			t.skip("unknown top-level block type", key)
		}
	}
}

// typedBlocks writes resource / data blocks: {"<type>": {"<name>": body}}.
func (t *jsonTranslator) typedBlocks(blockType string, val any) {
	t.forEachObject(blockType, val, func(types *jsonObject) {
		for i, kind := range types.keys {
			if kind == "//" {
				continue
			}
			if !validIdent(kind) {
				t.skip("block label is not a valid name", blockType+"."+kind)
				continue
			}
			t.forEachObject(blockType+"."+kind, types.vals[i], func(names *jsonObject) {
				for j, name := range names.keys {
					if name == "//" {
						continue
					}
					addr := blockType + "." + kind + "." + name
					if !validIdent(name) {
						t.skip("block label is not a valid name", addr)
						continue
					}
					t.forEachObject(addr, names.vals[j], func(body *jsonObject) {
						fmt.Fprintf(&t.b, "%s %q %q {\n", blockType, kind, name)
						t.attributes(body, bodyResource, addr)
						t.b.WriteString("}\n")
					})
				}
			})
		}
	})
}

// namedBlocks writes module / provider / variable blocks: {"<name>": body},
// where body may be an array (several provider configurations).
func (t *jsonTranslator) namedBlocks(blockType string, val any, kind bodyKind) {
	t.forEachObject(blockType, val, func(names *jsonObject) {
		for i, name := range names.keys {
			if name == "//" {
				continue
			}
			addr := blockType + "." + name
			if !validIdent(name) {
				t.skip("block label is not a valid name", addr)
				continue
			}
			t.forEachObject(addr, names.vals[i], func(body *jsonObject) {
				fmt.Fprintf(&t.b, "%s %q {\n", blockType, name)
				t.attributes(body, kind, addr)
				t.b.WriteString("}\n")
			})
		}
	})
}

// forEachObject calls fn for val when it is an object, or for each object
// in val when it is an array (Terraform JSON's form for repeated blocks).
func (t *jsonTranslator) forEachObject(where string, val any, fn func(*jsonObject)) {
	switch v := val.(type) {
	case *jsonObject:
		fn(v)
	case []any:
		for _, e := range v {
			if o, ok := e.(*jsonObject); ok {
				fn(o)
			} else {
				t.skip("expected an object", where)
			}
		}
	default:
		t.skip("expected an object", where)
	}
}

func (t *jsonTranslator) attributes(obj *jsonObject, kind bodyKind, where string) {
	for i, key := range obj.keys {
		val := obj.vals[i]
		addr := where + "." + key
		if key == "//" {
			continue
		}
		if !validIdent(key) {
			t.skip("argument name is not a valid name", addr)
			continue
		}
		switch kind {
		case bodyResource:
			switch key {
			case "depends_on", "lifecycle", "provisioner", "connection":
				continue
			case "provider":
				t.reference(key, val, addr)
				continue
			case "dynamic":
				t.dynamic(val, addr)
				continue
			}
		case bodyModule:
			switch key {
			case "depends_on":
				continue
			case "providers":
				t.providersMap(val, addr)
				continue
			}
		case bodyVariable:
			switch key {
			case "type":
				t.typeExpr(val, addr)
				continue
			case "validation":
				continue
			}
		}
		fmt.Fprintf(&t.b, "%s = ", key)
		t.value(val)
		t.b.WriteString("\n")
	}
}

// reference writes a string that Terraform JSON reads as a reference
// (provider = "aws.eu") as the bare reference.
func (t *jsonTranslator) reference(key string, val any, addr string) {
	s, ok := val.(string)
	if !ok || !referencePattern.MatchString(s) {
		t.skip("expected a provider reference such as \"aws.eu\"", addr)
		return
	}
	fmt.Fprintf(&t.b, "%s = %s\n", key, s)
}

func (t *jsonTranslator) providersMap(val any, addr string) {
	obj, ok := val.(*jsonObject)
	if !ok {
		t.skip("expected an object of provider references", addr)
		return
	}
	t.b.WriteString("providers = {\n")
	for i, k := range obj.keys {
		s, ok := obj.vals[i].(string)
		if !referencePattern.MatchString(k) || !ok || !referencePattern.MatchString(s) {
			t.skip("expected a provider reference such as \"aws.eu\"", addr+"."+k)
			continue
		}
		fmt.Fprintf(&t.b, "%s = %s\n", k, s)
	}
	t.b.WriteString("}\n")
}

// typeExpr writes a variable's type constraint, a string holding a type
// expression ("list(string)") in Terraform JSON.
func (t *jsonTranslator) typeExpr(val any, addr string) {
	s, ok := val.(string)
	if !ok || strings.ContainsAny(s, "\n\r#") || strings.Contains(s, "//") || strings.Contains(s, "/*") {
		t.skip("expected a type expression such as \"list(string)\"", addr)
		return
	}
	if _, diags := hclsyntax.ParseExpression([]byte(s), t.path, hcl.InitialPos); diags.HasErrors() {
		t.skip("expected a type expression such as \"list(string)\"", addr)
		return
	}
	fmt.Fprintf(&t.b, "type = %s\n", s)
}

// dynamic writes {"dynamic": {"<label>": {"for_each": ..., "iterator":
// "x", "content": {...}}}} as native dynamic blocks.
func (t *jsonTranslator) dynamic(val any, where string) {
	t.forEachObject(where, val, func(labels *jsonObject) {
		for i, label := range labels.keys {
			if label == "//" {
				continue
			}
			addr := where + "." + label
			if !validIdent(label) {
				t.skip("block label is not a valid name", addr)
				continue
			}
			t.forEachObject(addr, labels.vals[i], func(body *jsonObject) {
				fmt.Fprintf(&t.b, "dynamic %q {\n", label)
				for j, key := range body.keys {
					v := body.vals[j]
					switch key {
					case "for_each":
						t.b.WriteString("for_each = ")
						t.value(v)
						t.b.WriteString("\n")
					case "iterator":
						if s, ok := v.(string); ok && validIdent(s) {
							fmt.Fprintf(&t.b, "iterator = %s\n", s)
						} else {
							t.skip("expected an iterator name", addr+".iterator")
						}
					case "content":
						t.forEachObject(addr+".content", v, func(c *jsonObject) {
							t.b.WriteString("content {\n")
							t.attributes(c, bodyResource, addr)
							t.b.WriteString("}\n")
						})
					}
				}
				t.b.WriteString("}\n")
			})
		}
	})
}

// value writes a JSON value as a native expression.
func (t *jsonTranslator) value(v any) {
	switch x := v.(type) {
	case nil:
		t.b.WriteString("null")
	case bool:
		fmt.Fprintf(&t.b, "%t", x)
	case json.Number:
		t.b.WriteString(x.String())
	case string:
		t.b.WriteString(quoteTemplate(x))
	case []any:
		t.b.WriteString("[")
		for i, e := range x {
			if i > 0 {
				t.b.WriteString(", ")
			}
			t.value(e)
		}
		t.b.WriteString("]")
	case *jsonObject:
		t.b.WriteString("{\n")
		for i, k := range x.keys {
			// Object keys are literal strings in Terraform JSON.
			t.b.WriteString(quoteTemplate(escapeTemplate(k)))
			t.b.WriteString(" = ")
			t.value(x.vals[i])
			t.b.WriteString("\n")
		}
		t.b.WriteString("}")
	default:
		t.b.WriteString("null")
	}
}

// quoteTemplate writes s as a native quoted template. ${ and %{ sequences
// are kept: they are templates in Terraform JSON too.
func quoteTemplate(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// escapeTemplate makes s a template that evaluates to s literally.
func escapeTemplate(s string) string {
	s = strings.ReplaceAll(s, "${", "$${")
	return strings.ReplaceAll(s, "%{", "%%{")
}

var (
	identPattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	referencePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(\.[A-Za-z_][A-Za-z0-9_-]*)?$`)
)

func validIdent(s string) bool { return identPattern.MatchString(s) }
