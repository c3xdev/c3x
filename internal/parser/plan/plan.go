// Package plan parses Terraform plan JSON (`terraform show -json`) into
// domain.Resources. Plan files already carry resolved attribute values
// so the parser is straightforward decoding plus address normalization;
// none of the HCL evaluator machinery is involved.
package plan

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
)

// ParseFile reads a plan.json file and returns the resources it
// describes. The estimate reflects the post-apply state: every resource
// that will exist after the plan is applied, priced as a whole (the same
// meaning as estimating the .tf), not just the resources this plan
// changes. Use `--show-delta` / `c3x diff` for the change-only view.
func ParseFile(path string, logger *slog.Logger) ([]domain.Resource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseBytes(raw, logger)
}

// ParseBytes parses an in-memory plan JSON document. Exported so tests
// and pipes can drive the parser without a temp file.
//
// The resource set comes from planned_values (the complete post-apply
// state) so unchanged resources are priced too — a plan where nothing
// changes still reports the full cost of the infrastructure it
// describes. Older plan formats that omit planned_values fall back to
// resource_changes, excluding resources scheduled purely for destruction
// (they won't exist post-apply).
func ParseBytes(raw []byte, logger *slog.Logger) ([]domain.Resource, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var doc planFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode plan: %w", err)
	}

	region := defaultRegion(doc.Configuration)
	out := make([]domain.Resource, 0, len(doc.ResourceChanges))

	// Build action lookup from resource_changes so we can annotate
	// each resource with what Terraform intends to do with it.
	actions := buildActionMap(doc.ResourceChanges)

	// Primary: planned_values is the full desired (post-apply) state.
	if doc.PlannedValues != nil {
		collectPlanned(doc.PlannedValues.RootModule, region, actions, &out)
	}

	// Fallback for plan formats that omit planned_values.
	if len(out) == 0 {
		for _, rc := range doc.ResourceChanges {
			if isDeleteOnly(rc.Change.Actions) {
				continue
			}
			attrs, _ := rc.Change.After.(map[string]any)
			r := domain.Resource{
				Ref:        domain.Reference{Kind: rc.Type, Name: nameFromAddress(rc.Address, rc.Name)},
				Attributes: attrs,
				Action:     classifyActions(rc.Change.Actions),
			}
			if region != "" {
				rgn := region
				r.Region = &rgn
			}
			out = append(out, r)
		}
	}

	// Append resources scheduled for pure deletion. These don't exist
	// in planned_values (they won't be in the post-apply state) but
	// the delta renderer needs them to show what's being removed.
	// We use the "before" attributes so the calculator can price them.
	for _, rc := range doc.ResourceChanges {
		if isDeleteOnly(rc.Change.Actions) {
			attrs, _ := rc.Change.Before.(map[string]any)
			r := domain.Resource{
				Ref:        domain.Reference{Kind: rc.Type, Name: nameFromAddress(rc.Address, rc.Name)},
				Attributes: attrs,
				Action:     domain.PlanActionDelete,
			}
			if region != "" {
				rgn := region
				r.Region = &rgn
			}
			out = append(out, r)
		}
	}

	logger.Debug("plan parsed", "resources", len(out), "region", region)
	return out, nil
}

// collectPlanned walks a planned_values module tree, appending every
// managed resource. Data sources and null child-module entries (which
// untrusted plan JSON can contain) are skipped rather than dereferenced.
// Each resource is annotated with its PlanAction from the actions map.
func collectPlanned(mod *plannedModule, region string, actions map[string]domain.PlanAction, out *[]domain.Resource) {
	if mod == nil {
		return
	}
	for _, pr := range mod.Resources {
		if pr.Mode != "managed" {
			continue
		}
		attrs, _ := pr.Values.(map[string]any)
		r := domain.Resource{
			Ref:        domain.Reference{Kind: pr.Type, Name: nameFromAddress(pr.Address, pr.Name)},
			Attributes: attrs,
			Action:     actions[pr.Address],
		}
		if region != "" {
			rgn := region
			r.Region = &rgn
		}
		*out = append(*out, r)
	}
	for _, child := range mod.ChildModules {
		collectPlanned(child, region, actions, out)
	}
}

// buildActionMap creates an address → PlanAction lookup from
// resource_changes. This lets planned_values resources carry the
// intent metadata (create/update/no-op) without losing completeness.
func buildActionMap(changes []resourceChange) map[string]domain.PlanAction {
	m := make(map[string]domain.PlanAction, len(changes))
	for _, rc := range changes {
		m[rc.Address] = classifyActions(rc.Change.Actions)
	}
	return m
}

// classifyActions reduces Terraform's actions array to a single
// PlanAction. Terraform uses combinations like ["delete","create"] for
// replacements; we collapse those to the most relevant single action.
func classifyActions(actions []string) domain.PlanAction {
	if len(actions) == 0 {
		return domain.PlanActionNoOp
	}
	if len(actions) == 1 {
		switch actions[0] {
		case "create":
			return domain.PlanActionCreate
		case "delete":
			return domain.PlanActionDelete
		case "update":
			return domain.PlanActionUpdate
		case "no-op", "read":
			return domain.PlanActionNoOp
		}
	}
	// Multi-action: ["delete","create"] = replace, ["create","delete"] = replace
	// Treat any combo containing "create" or "update" as an update.
	for _, a := range actions {
		if a == "create" || a == "update" {
			return domain.PlanActionUpdate
		}
	}
	return domain.PlanActionNoOp
}

// nameFromAddress reconstructs the resource name preserving any
// `module.X.module.Y.` prefix. Terraform addresses look like
// `module.outer.module.inner.aws_instance.web` — we strip the trailing
// `<kind>.<name>` segments and keep everything before so the .tf path
// and the plan-JSON path produce identical references downstream.
func nameFromAddress(address, fallback string) string {
	suffix := "." + fallback
	if idx := strings.LastIndex(address, suffix); idx >= 0 {
		head := address[:idx]
		if dot := strings.LastIndex(head, "."); dot >= 0 {
			prefix := head[:dot]
			if prefix == "" {
				return fallback
			}
			return prefix + "." + fallback
		}
		return fallback
	}
	// Defensive fallback: keep the last segment.
	parts := strings.Split(address, ".")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return fallback
	}
	return parts[len(parts)-1]
}

// isDeleteOnly reports whether a resource_changes entry is scheduled
// purely for destruction, so it won't exist post-apply. Replaces
// (delete+create) and no-op resources are kept.
func isDeleteOnly(actions []string) bool {
	if len(actions) == 0 {
		return false
	}
	for _, a := range actions {
		if a != "delete" {
			return false
		}
	}
	return true
}

func defaultRegion(cfg *configuration) string {
	if cfg == nil {
		return ""
	}
	// First match wins: AWS, Azure, GCP, GCP-beta.
	for _, p := range []struct {
		Provider string
		Attr     string
	}{
		{"aws", "region"},
		{"azurerm", "location"},
		{"google", "region"},
		{"google-beta", "region"},
	} {
		pc, ok := cfg.ProviderConfig[p.Provider]
		if !ok {
			continue
		}
		raw, ok := pc.Expressions[p.Attr]
		if !ok {
			continue
		}
		expr, ok := parseExpression(raw)
		if !ok {
			continue
		}
		if expr.ConstantValue != nil {
			if s, ok := expr.ConstantValue.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// planFile mirrors the fields we care about from `terraform show -json`.
// Anything not declared here is ignored at decode time.
type planFile struct {
	PlannedValues   *plannedValues   `json:"planned_values"`
	ResourceChanges []resourceChange `json:"resource_changes"`
	Configuration   *configuration   `json:"configuration"`
}

// plannedValues is the projected post-apply state.
type plannedValues struct {
	RootModule *plannedModule `json:"root_module"`
}

type plannedModule struct {
	Resources    []plannedResource `json:"resources"`
	ChildModules []*plannedModule  `json:"child_modules"`
}

type plannedResource struct {
	Address string `json:"address"`
	Mode    string `json:"mode"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Values  any    `json:"values"`
}

type resourceChange struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Change  change `json:"change"`
}

type change struct {
	Actions []string `json:"actions"`
	Before  any      `json:"before"`
	After   any      `json:"after"`
}

type configuration struct {
	ProviderConfig map[string]providerConfig `json:"provider_config"`
}

type providerConfig struct {
	Expressions map[string]json.RawMessage `json:"expressions"`
}

// expression is the loose shape Terraform uses for both constants and
// references. We only read constant_value; complex expressions left as
// references stay unresolved.
type expression struct {
	ConstantValue any `json:"constant_value"`
}

// parseExpression attempts to decode a raw JSON value as an expression
// object. Terraform's plan JSON can store expressions as either objects
// ({"constant_value": ...}) or arrays (e.g. the `features` block in
// azurerm). We only care about the object form with constant_value.
func parseExpression(raw json.RawMessage) (expression, bool) {
	var expr expression
	if err := json.Unmarshal(raw, &expr); err != nil {
		return expression{}, false
	}
	return expr, true
}
