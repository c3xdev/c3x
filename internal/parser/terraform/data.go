package terraform

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// collectDataBlocks pre-scans every `data "kind" "name" { ... }` block
// and builds a `data` cty.Object whose shape mirrors Terraform's own
// scope, so `data.aws_ami.ubuntu.id` traversals resolve at evaluation
// time. c3x doesn't call AWS/Azure/GCP APIs to materialise data sources,
// so each block exposes:
//
//   - `id`, as the synthetic string "data.<kind>.<name>.id";
//   - the static literal attributes the block declares in its body;
//   - for the handful of data sources whose results drive resource counts
//     in common configurations (the availability zones of a region, the
//     current region, account and project), a placeholder with a
//     plausible value. Placeholders are marked (see placeholderMark), so
//     every value computed from one is traceable: a resource whose count
//     or for_each depends on a placeholder is logged as a warning, and an
//     attribute computed from one is reported unresolved, as it was before
//     placeholders existed.
//
// regions supplies the region of the provider each data block uses, so
// the availability zones of an eu-west-1 provider are eu-west-1a..c.
func collectDataBlocks(sources []sourceFile, regions providerRegions) cty.Value {
	byKind := map[string]map[string]cty.Value{}

	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "data" || len(block.Labels) < 2 {
				continue
			}
			kind := block.Labels[0]
			name := block.Labels[1]

			fields := map[string]cty.Value{
				"id": cty.StringVal(fmt.Sprintf("data.%s.%s.id", kind, name)),
			}
			literals := map[string]cty.Value{}
			ctx := buildEvalContext(cty.EmptyObjectVal, cty.EmptyObjectVal, cty.EmptyObjectVal, nil)
			for _, attr := range block.Body.Attributes {
				val, diags := attr.Expr.Value(ctx)
				if diags.HasErrors() {
					continue
				}
				literals[attr.Name] = val
			}
			region := dataRegion(kind, block.Body, regions, literals)
			for k, v := range dataPlaceholders(kind, name, region) {
				fields[k] = v
			}
			// What the configuration states beats what c3x assumes.
			for k, v := range literals {
				fields[k] = v
			}

			if byKind[kind] == nil {
				byKind[kind] = map[string]cty.Value{}
			}
			byKind[kind][name] = cty.ObjectVal(fields)
		}
	}

	if len(byKind) == 0 {
		return cty.EmptyObjectVal
	}
	top := map[string]cty.Value{}
	for kind, names := range byKind {
		top[kind] = cty.ObjectVal(names)
	}
	return cty.ObjectVal(top)
}

// placeholderMark is attached (as a cty mark) to every placeholder value a
// data source exposes. cty carries marks through every operation and
// function, so a value derived from a placeholder, however indirectly
// (a local slicing the zone list, a module input, a for expression),
// still carries it when a resource reads it.
type placeholderMark struct {
	// Ref is the data source attribute: data.aws_availability_zones.available.names.
	Ref string
	// Assumed is the placeholder value, rendered for the user.
	Assumed string
}

// dataPlaceholders returns the placeholder attributes for one data
// source, or nil when c3x has none for its kind.
func dataPlaceholders(kind, name, region string) map[string]cty.Value {
	ref := "data." + kind + "." + name + "."
	out := map[string]cty.Value{}
	set := func(attr string, v cty.Value) {
		out[attr] = v.Mark(placeholderMark{Ref: ref + attr, Assumed: renderPlaceholder(v)})
	}
	strs := func(ss ...string) cty.Value {
		vals := make([]cty.Value, len(ss))
		for i, s := range ss {
			vals[i] = cty.StringVal(s)
		}
		return cty.ListVal(vals)
	}
	const (
		awsAccount   = "123456789012"
		azureUUID    = "00000000-0000-0000-0000-000000000000"
		googleProjID = "placeholder-project"
	)
	switch kind {
	case "aws_availability_zones":
		r := orDefault(region, "us-east-1")
		set("names", strs(r+"a", r+"b", r+"c"))
		set("zone_ids", strs(r+"-az1", r+"-az2", r+"-az3"))
	case "aws_region":
		r := orDefault(region, "us-east-1")
		set("name", cty.StringVal(r))
		set("region", cty.StringVal(r))
		set("id", cty.StringVal(r))
	case "aws_caller_identity":
		set("account_id", cty.StringVal(awsAccount))
		set("arn", cty.StringVal("arn:aws:iam::"+awsAccount+":root"))
		set("user_id", cty.StringVal(awsAccount))
	case "aws_partition":
		set("partition", cty.StringVal("aws"))
		set("dns_suffix", cty.StringVal("amazonaws.com"))
	case "google_client_config":
		r := orDefault(region, "us-central1")
		set("region", cty.StringVal(r))
		set("zone", cty.StringVal(r+"-a"))
		set("project", cty.StringVal(googleProjID))
	case "google_compute_zones":
		r := orDefault(region, "us-central1")
		set("names", strs(r+"-a", r+"-b", r+"-c"))
	case "google_project":
		set("project_id", cty.StringVal(googleProjID))
		set("number", cty.StringVal("000000000000"))
	case "azurerm_client_config":
		for _, a := range []string{"client_id", "tenant_id", "subscription_id", "object_id"} {
			set(a, cty.StringVal(azureUUID))
		}
	case "azurerm_subscription":
		set("subscription_id", cty.StringVal(azureUUID))
		set("tenant_id", cty.StringVal(azureUUID))
		set("display_name", cty.StringVal("placeholder-subscription"))
	default:
		return nil
	}
	return out
}

// dataRegion is the region of the provider a data block reads from: its
// `provider = aws.eu` reference, else the default configuration of its
// provider. aws_region's own `name`/`region` argument selects a region
// explicitly and wins. Empty when unknown; the placeholder then uses the
// cloud's reference region.
func dataRegion(kind string, body *hclsyntax.Body, regions providerRegions, literals map[string]cty.Value) string {
	if kind == "aws_region" {
		for _, a := range []string{"name", "region"} {
			if v, ok := literals[a]; ok && v.Type() == cty.String && v.IsKnown() && !v.IsNull() {
				return v.AsString()
			}
		}
	}
	if attr := body.Attributes["provider"]; attr != nil {
		if r, ok := regions.fromReference(attr.Expr, &hcl.EvalContext{}); ok {
			return r
		}
	}
	return regions.byKey[providerForKind(kind)]
}

func renderPlaceholder(v cty.Value) string {
	if v.Type() == cty.String {
		return v.AsString()
	}
	var parts []string
	for it := v.ElementIterator(); it.Next(); {
		_, e := it.Element()
		parts = append(parts, e.AsString())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// stripPlaceholders returns v with every mark removed, plus the
// placeholders v was computed from. Go code must call it before reading a
// value (AsString, ElementIterator, ...), which panic on marked values.
func stripPlaceholders(v cty.Value) (cty.Value, []placeholderMark) {
	if v.Type() == cty.NilType || !v.ContainsMarked() {
		return v, nil
	}
	plain, marks := v.UnmarkDeep()
	var out []placeholderMark
	for m := range marks {
		if p, ok := m.(placeholderMark); ok {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return plain, out
}

// withPlaceholders marks v as derived from deps.
func withPlaceholders(v cty.Value, deps []placeholderMark) cty.Value {
	for _, d := range deps {
		v = v.Mark(d)
	}
	return v
}

// warnPlaceholders reports that how many instances of address exist
// (what: "count", "for_each") rests on placeholder data source values.
func warnPlaceholders(logger *slog.Logger, what, address string, deps []placeholderMark) {
	assumed := make([]string, len(deps))
	for i, d := range deps {
		assumed[i] = d.Ref + " = " + d.Assumed
	}
	logger.Warn(what+" depends on a data source c3x cannot query; it assumed a placeholder value, "+
		"so the number of instances priced may differ from what Terraform would create "+
		"(price a plan JSON for exact results)",
		"resource", address, "assumed", strings.Join(assumed, "; "))
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
