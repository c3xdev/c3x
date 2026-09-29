package cloudformation

import (
	"regexp"
	"strconv"
	"strings"
)

// numericString matches a plain decimal number: an optional minus sign,
// digits without a leading zero, and an optional fraction. Exponents,
// hex, and leading "+" are left as strings.
var numericString = regexp.MustCompile(`^-?(0|[1-9]\d*)(\.\d+)?$`)

// coerceScalars converts numeric and boolean strings to the numbers and
// booleans they spell, recursively through maps and lists.
//
// CloudFormation accepts a string wherever a property takes a number or
// a boolean (`AllocatedStorage: "250"`, `MultiAZ: "true"`), and every
// parameter value is a string once it is referenced (`Type: Number`
// parameters included, and `--var` overrides always). The catalog
// expressions do arithmetic and comparisons on those values, which
// fail on a string, so the parser hands them over typed.
//
// Only strings that read back identically are converted: "250" and
// "16.4" become numbers, while "8.0", "007" or "1e3" stay strings, so
// a version or identifier that happens to look numeric keeps its exact
// spelling when a catalog filter stringifies it again.
func coerceScalars(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = coerceScalars(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = coerceScalars(val)
		}
		return t
	case string:
		return coerceString(t)
	}
	return v
}

func coerceString(s string) any {
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	if !numericString.MatchString(s) {
		return s
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || strconv.FormatFloat(f, 'f', -1, 64) != s {
		return s
	}
	return f
}
