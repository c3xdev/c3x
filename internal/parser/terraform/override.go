package terraform

import (
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// mergeOverrides applies override files (override.tf, *_override.tf and
// their .tf.json / .tofu / .tofu.json forms) to the primary files, with
// Terraform's merge rules, in the lexical order Terraform applies them:
//
//   - each top-level block in an override file is merged into the primary
//     block with the same type and labels (for provider blocks: the same
//     name and alias); an override block with no original is ignored with
//     a warning, where Terraform rejects the configuration;
//   - an attribute in the override replaces the original's attribute of
//     the same name;
//   - a nested block in the override replaces every nested block of the
//     same type in the original (a `dynamic "x"` block counts as type x),
//     except `lifecycle`, whose arguments merge one by one;
//   - a locals entry replaces the original entry of the same name,
//     whichever locals block holds it.
//
// Before this, override files loaded as ordinary files, so an overridden
// resource was priced twice, once with each configuration.
//
// The primary ASTs are modified in place; each load parses fresh.
func mergeOverrides(primary, overrides []sourceFile, logger *slog.Logger) {
	for _, ov := range overrides {
		for _, ob := range ov.Body.Blocks {
			if ob.Type == "locals" {
				mergeLocals(primary, ob, ov.Path, logger)
				continue
			}
			target := findOverridden(primary, ob)
			if target == nil && ob.Type == "terraform" {
				continue // settings only; nothing c3x prices
			}
			if target == nil {
				logger.Warn("override block has no original block to override; ignored",
					"file", ov.Path, "block", blockAddress(ob))
				continue
			}
			mergeBody(target.Body, ob.Body, target.Type == "resource" || target.Type == "data")
		}
	}
}

// findOverridden returns the primary block ob overrides, or nil.
func findOverridden(primary []sourceFile, ob *hclsyntax.Block) *hclsyntax.Block {
	key := blockKey(ob)
	for _, src := range primary {
		for _, b := range src.Body.Blocks {
			if b.Type == ob.Type && blockKey(b) == key {
				return b
			}
		}
	}
	return nil
}

// blockKey identifies a top-level block for override matching: its type
// and labels, plus the alias for provider blocks, since `provider "aws"`
// with alias = "eu" is a different configuration from the default one.
func blockKey(b *hclsyntax.Block) string {
	key := b.Type + "\x00" + strings.Join(b.Labels, "\x00")
	if b.Type == "provider" {
		if attr := b.Body.Attributes["alias"]; attr != nil {
			if v, diags := attr.Expr.Value(nil); !diags.HasErrors() && v.Type() == cty.String && v.IsKnown() && !v.IsNull() {
				key += "\x00alias=" + v.AsString()
			}
		}
	}
	return key
}

func blockAddress(b *hclsyntax.Block) string {
	return strings.Join(append([]string{b.Type}, b.Labels...), ".")
}

// mergeBody merges an override body into dst. mergeLifecycle selects the
// argument-by-argument lifecycle rule that resource and data blocks use.
func mergeBody(dst, src *hclsyntax.Body, mergeLifecycle bool) {
	if dst.Attributes == nil {
		dst.Attributes = hclsyntax.Attributes{}
	}
	for name, attr := range src.Attributes {
		dst.Attributes[name] = attr
	}

	replaced := map[string]bool{}
	var added []*hclsyntax.Block
	for _, b := range src.Blocks {
		t := effectiveBlockType(b)
		if mergeLifecycle && t == "lifecycle" {
			if existing := firstBlockOfType(dst, "lifecycle"); existing != nil {
				mergeBody(existing.Body, b.Body, false)
				continue
			}
		}
		replaced[t] = true
		added = append(added, b)
	}
	if len(replaced) == 0 {
		return
	}
	merged := make(hclsyntax.Blocks, 0, len(dst.Blocks)+len(added))
	for _, b := range dst.Blocks {
		if !replaced[effectiveBlockType(b)] {
			merged = append(merged, b)
		}
	}
	merged = append(merged, added...)
	dst.Blocks = merged
}

// effectiveBlockType is the block type a nested block produces: its type,
// or for `dynamic "x"` the x it generates.
func effectiveBlockType(b *hclsyntax.Block) string {
	if b.Type == "dynamic" && len(b.Labels) == 1 {
		return b.Labels[0]
	}
	return b.Type
}

func firstBlockOfType(body *hclsyntax.Body, t string) *hclsyntax.Block {
	for _, b := range body.Blocks {
		if b.Type == t {
			return b
		}
	}
	return nil
}

// mergeLocals replaces each local the override block defines in whichever
// primary locals block defines it.
func mergeLocals(primary []sourceFile, ob *hclsyntax.Block, path string, logger *slog.Logger) {
	for name, attr := range ob.Body.Attributes {
		found := false
		for _, src := range primary {
			for _, b := range src.Body.Blocks {
				if b.Type != "locals" {
					continue
				}
				if _, ok := b.Body.Attributes[name]; ok {
					b.Body.Attributes[name] = attr
					found = true
				}
			}
		}
		if !found {
			logger.Warn("override file defines a local that no other file defines; ignored",
				"file", path, "local", name)
		}
	}
}
