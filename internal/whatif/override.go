// Package whatif provides resource attribute override functionality for cost scenario analysis.
package whatif

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/tidwall/gjson"
)

// Override represents a single resource attribute override.
type Override struct {
	ResourceAddress string
	AttributePath   string
	Value           string
}

// ParseOverride parses a what-if override string in the format
// "resource_type.name.attribute=value" or "module.x.resource_type.name.attribute=value".
// The attribute is always the last dot-separated segment of the path.
func ParseOverride(s string) (*Override, error) {
	eqIdx := strings.Index(s, "=")
	if eqIdx < 0 {
		return nil, fmt.Errorf("invalid what-if format %q: expected 'resource_type.name.attribute=value'", s)
	}

	path := s[:eqIdx]
	value := s[eqIdx+1:]

	// Split from the right: the attribute is the last segment, everything
	// before it is the resource address. This supports module-scoped addresses
	// like "module.vpc.aws_instance.web.instance_type".
	lastDot := strings.LastIndex(path, ".")
	if lastDot <= 0 {
		return nil, fmt.Errorf("invalid what-if path %q: expected 'resource_type.name.attribute'", path)
	}

	return &Override{
		ResourceAddress: path[:lastDot],
		AttributePath:   path[lastDot+1:],
		Value:           value,
	}, nil
}

// ApplyOverrides modifies resource metadata in projects before resources are built.
// Returns the number of overrides successfully applied.
func ApplyOverrides(projects []*engine.Workspace, overrides []*Override) (int, error) {
	applied := 0

	for _, project := range projects {
		for _, pr := range project.PartialResources {
			for _, o := range overrides {
				if matchesAddress(pr.Address, o.ResourceAddress) {
					if raw, ok := pr.Metadata["values"]; ok {
						modified := setJSONValue(raw.Raw, o.AttributePath, o.Value)
						pr.Metadata["values"] = gjson.Parse(modified)
						applied++
					}
				}
			}
		}
	}

	return applied, nil
}

func matchesAddress(resourceAddr, overrideAddr string) bool {
	if resourceAddr == overrideAddr {
		return true
	}
	return strings.HasSuffix(resourceAddr, "."+overrideAddr)
}

// setJSONValue sets a key in a JSON object to a new value, preserving all
// other keys. The value is type-coerced (bool, int, float) so overrides like
// multi_az=true produce JSON true, not "true".
func setJSONValue(rawJSON, key, value string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &m); err != nil {
		if rawJSON == "" || rawJSON == "{}" {
			m = make(map[string]interface{}) // empty input: start fresh
		} else {
			return rawJSON // non-object input (array, number, malformed): preserve original
		}
	}
	m[key] = coerceValue(value)
	b, err := json.Marshal(m)
	if err != nil {
		return rawJSON // preserve original on marshal failure
	}
	return string(b)
}

// coerceValue attempts to parse a string as bool, int, or float before
// falling back to string. This ensures what-if overrides produce correctly
// typed JSON values (e.g., "true" → true, "10" → 10).
func coerceValue(s string) interface{} {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
