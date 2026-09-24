package terraform

import (
	"log/slog"

	"github.com/zclconf/go-cty/cty"
)

// findDefaultRegion walks every `provider "<cloud>" { ... }` block and
// returns the first region/location value that resolves successfully.
// The lookup is multi-cloud: AWS uses `region`, Azure uses `location`,
// GCP uses `region` (and `google-beta` is treated like `google`).
//
// Returning an empty string means "no provider block had a usable
// region"; the calculator's defaultRegion (from the config layer) then
// fills the gap.
func findDefaultRegion(
	sources []sourceFile,
	vars map[string]cty.Value,
	locals map[string]cty.Value,
	data cty.Value,
	logger *slog.Logger,
) string {
	for _, src := range sources {
		for _, block := range src.Body.Blocks {
			if block.Type != "provider" || len(block.Labels) == 0 {
				continue
			}
			attrName := providerRegionAttr(block.Labels[0])
			if attrName == "" {
				continue
			}
			attr, ok := block.Body.Attributes[attrName]
			if !ok {
				continue
			}
			ctx := buildEvalContext(asObject(vars), asObject(locals), data, nil)
			val, diags := attr.Expr.Value(ctx)
			val, _ = stripPlaceholders(val)
			if diags.HasErrors() {
				logger.Debug("provider region attr evaluation failed",
					"provider", block.Labels[0],
					"file", src.Path,
					"diags", formatDiags(diags))
				continue
			}
			if val.Type() == cty.String && !val.IsNull() && val.IsKnown() {
				return val.AsString()
			}
		}
	}
	return ""
}

func providerRegionAttr(provider string) string {
	switch provider {
	case "aws":
		return "region"
	case "azurerm":
		return "location"
	case "google", "google-beta":
		return "region"
	default:
		return ""
	}
}
