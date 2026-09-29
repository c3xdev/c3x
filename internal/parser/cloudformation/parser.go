package cloudformation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"gopkg.in/yaml.v3"
)

// Options carry caller-supplied overrides parallel to those the
// Terraform parser accepts.
type Options struct {
	// Parameters overrides CFN parameter values that would otherwise
	// be filled in from the template's `Default` fields. These come
	// from CLI `--var name=value` flags (the same flag c3x uses for
	// tfvars; CFN templates are language-agnostic).
	Parameters map[string]any
	// Region is the deployment region used to populate the
	// AWS::Region pseudo-parameter and the resulting domain.Resource
	// region pointer. Empty means "rely on config.Resolved.Region".
	Region string
	Logger *slog.Logger
}

// ParseFile detects whether the file is JSON or YAML by extension
// (and falls back to content sniff if the extension is ambiguous),
// then runs the format-specific parser.
func ParseFile(path string, opts Options) ([]domain.Resource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseBytes(raw, path, opts)
}

// ParseBytes is the in-memory entry point. Exported so tests don't
// need temp files.
func ParseBytes(raw []byte, originPath string, opts Options) ([]domain.Resource, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	// Decode as YAML first; a JSON document is a strict subset of YAML
	// 1.2 so the same decoder handles both. We could test for { vs
	// non-{ first, but yaml.Unmarshal on JSON content works correctly
	// and avoids two code paths.
	template, err := decodeTemplate(raw, originPath)
	if err != nil {
		return nil, err
	}

	sc := scope{
		Parameters: resolveParameters(template.Parameters, opts.Parameters),
		Mappings:   normalisedMappings(template.Mappings),
		Pseudo: map[string]any{
			"AWS::Region":           nonEmpty(opts.Region, "us-east-1"),
			"AWS::AccountId":        "<account-id>",
			"AWS::Partition":        "aws",
			"AWS::URLSuffix":        "amazonaws.com",
			"AWS::NoValue":          nil,
			"AWS::StackName":        "<stack>",
			"AWS::StackId":          "<stack-id>",
			"AWS::NotificationARNs": []any{},
		},
	}

	var out []domain.Resource
	for logicalID, res := range template.Resources {
		kind := translateKind(res.Type)
		if kind == "" {
			opts.Logger.Debug("cloudformation: skipping unknown type",
				"logical_id", logicalID, "type", res.Type)
			continue
		}
		resolved, ok := coerceScalars(resolve(res.Properties, sc)).(map[string]any)
		if !ok {
			resolved = map[string]any{}
		}
		attrs := translateProps(kind, resolved)
		// Stamp the OS attribute for compute resources where the CFN
		// template doesn't declare one (Linux is the safe default).
		ensureOSAttr(kind, attrs)

		region := opts.Region
		emitted := domain.Resource{
			Ref:        domain.Reference{Kind: kind, Name: logicalID},
			Attributes: attrs,
		}
		if region != "" {
			emitted.Region = &region
		}
		out = append(out, emitted)
	}
	return out, nil
}

// template mirrors the relevant top-level CFN sections. We don't
// bind every field — the goformation library does, at the cost of
// hundreds of generated files, and we don't need that depth.
type template struct {
	FormatVersion any                                  `yaml:"AWSTemplateFormatVersion" json:"AWSTemplateFormatVersion"`
	Parameters    map[string]parameterDef              `yaml:"Parameters" json:"Parameters"`
	Mappings      map[string]map[string]map[string]any `yaml:"Mappings" json:"Mappings"`
	Resources     map[string]resourceDef               `yaml:"Resources" json:"Resources"`
}

type parameterDef struct {
	Type    string `yaml:"Type" json:"Type"`
	Default any    `yaml:"Default" json:"Default"`
}

type resourceDef struct {
	Type       string         `yaml:"Type" json:"Type"`
	Properties map[string]any `yaml:"Properties" json:"Properties"`
}

func decodeTemplate(raw []byte, originPath string) (*template, error) {
	t := &template{}
	if looksLikeJSON(raw) {
		if err := json.Unmarshal(raw, t); err != nil {
			return nil, fmt.Errorf("parsing %s as JSON: %w", originPath, err)
		}
		return t, nil
	}
	// YAML decode goes through an AST pass so we can expand the
	// short-form intrinsic tags (`!Ref`, `!Sub`, `!Join`, etc.) into
	// their long-form `Fn::*` map equivalents — the rest of the
	// parser only knows the long form.
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", originPath, err)
	}
	expandIntrinsicTags(&root)
	if err := root.Decode(t); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", originPath, err)
	}
	return t, nil
}

// expandIntrinsicTags rewrites a parsed YAML tree so that any node
// carrying a short-form CFN tag (`!Ref Size`) becomes the equivalent
// long-form mapping (`{Ref: Size}`). This is the only place tag
// handling lives — the rest of the parser sees clean Fn::* maps.
//
// We DON'T touch the YAML 1.2 standard tags (`!!str`, `!!int`, …);
// those start with `!!` and the decoder handles them natively.
func expandIntrinsicTags(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Tag != "" && strings.HasPrefix(n.Tag, "!") && !strings.HasPrefix(n.Tag, "!!") {
		switch n.Tag {
		case "!Ref", "!Condition":
			rewrap(n, strings.TrimPrefix(n.Tag, "!"))
		case "!Sub", "!Join", "!GetAtt", "!FindInMap",
			"!Select", "!Split", "!Base64", "!Cidr",
			"!ImportValue", "!Transform", "!Length",
			"!If", "!Equals", "!And", "!Or", "!Not":
			rewrap(n, "Fn::"+strings.TrimPrefix(n.Tag, "!"))
		default:
			// Unknown tag — leave alone. CFN custom intrinsics or
			// unrelated YAML tags pass through under their original
			// shape.
		}
	}
	// Special-case !GetAtt's shorthand string form (`!GetAtt Foo.Bar`)
	// which becomes `Fn::GetAtt: [Foo, Bar]` in the long form.
	if n.Kind == yaml.MappingNode && len(n.Content) == 2 {
		if key := n.Content[0]; key.Value == "Fn::GetAtt" {
			val := n.Content[1]
			if val.Kind == yaml.ScalarNode {
				parts := strings.SplitN(val.Value, ".", 2)
				val.Kind = yaml.SequenceNode
				val.Style = 0
				val.Value = ""
				val.Tag = ""
				val.Content = make([]*yaml.Node, 0, len(parts))
				for _, p := range parts {
					val.Content = append(val.Content, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: p,
					})
				}
			}
		}
	}
	for _, c := range n.Content {
		expandIntrinsicTags(c)
	}
}

// rewrap turns the receiver node (e.g. `!Ref Size`) into a mapping
// node containing `{key: <copy of receiver with tag stripped>}`. This
// is how we convert short-form tags into long-form maps without
// re-parsing.
func rewrap(n *yaml.Node, key string) {
	inner := &yaml.Node{
		Kind:    n.Kind,
		Style:   n.Style,
		Tag:     "",
		Value:   n.Value,
		Content: n.Content,
	}
	n.Kind = yaml.MappingNode
	n.Tag = ""
	n.Value = ""
	n.Style = 0
	n.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: key},
		inner,
	}
}

func looksLikeJSON(raw []byte) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		}
		return false
	}
	return false
}

// resolveParameters applies the user-supplied overrides on top of
// the template's Defaults. The result is what every Ref-to-parameter
// will see.
func resolveParameters(defs map[string]parameterDef, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(defs))
	for name, p := range defs {
		if p.Default != nil {
			out[name] = p.Default
		}
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// normalisedMappings returns a Mappings table with nil safety; CFN
// optionally omits the section entirely.
func normalisedMappings(in map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	if in == nil {
		return map[string]map[string]map[string]any{}
	}
	return in
}

// ensureOSAttr stamps `operating_system = "Linux"` on resource kinds
// where the catalog mapping requires it but CFN doesn't carry the
// field. Without this, aws_instance from CloudFormation reaches the
// calculator with no operatingSystem attribute and the catalog's
// `const = "Linux"` filter is the one that fires anyway. Belt-and-
// braces for catalogs that switch to `expr =`-driven filters later.
func ensureOSAttr(kind string, attrs map[string]any) {
	if kind == "aws_instance" {
		if _, ok := attrs["operating_system"]; !ok {
			attrs["operating_system"] = "Linux"
		}
	}
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// IsTemplate reports whether raw is a CloudFormation template: a
// document with a non-empty Resources section that either declares
// AWSTemplateFormatVersion or has at least one resource of an AWS::
// type. Used to pick templates out of a directory, where YAML and JSON
// files can be anything (usage files, CI config, Kubernetes manifests).
func IsTemplate(raw []byte) bool {
	t, err := decodeTemplate(raw, "")
	if err != nil || len(t.Resources) == 0 {
		return false
	}
	if t.FormatVersion != nil {
		return true
	}
	for _, r := range t.Resources {
		if strings.HasPrefix(r.Type, "AWS::") {
			return true
		}
	}
	return false
}
