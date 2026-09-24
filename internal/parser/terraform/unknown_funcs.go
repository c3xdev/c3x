package terraform

import (
	"log/slog"
	"regexp"
	"sync"

	"github.com/hashicorp/hcl/v2"
)

// unknownFuncName pulls the name out of HCL's "Call to unknown function"
// diagnostic detail: `There is no function named "file".`
var unknownFuncName = regexp.MustCompile(`function named "([^"]+)"`)

// warnedUnknown records which functions have already been reported, so a
// config that calls file() in forty resources warns once, not forty times.
var warnedUnknown sync.Map

// warnUnknownFunctions reports calls to functions the evaluator does not
// implement. Most unresolvable values are expected and stay quiet (a
// value only known after apply, an optional input nobody set), but an
// unknown function is never the user's intent. Left silent, it prices an
// attribute at its catalog default or, inside a local, drops every
// resource that depends on it. where names the attribute or local.
func warnUnknownFunctions(diags hcl.Diagnostics, logger *slog.Logger, where string) {
	for _, d := range diags {
		if d.Summary != "Call to unknown function" {
			continue
		}
		name := "?"
		if m := unknownFuncName.FindStringSubmatch(d.Detail); m != nil {
			name = m[1]
		}
		if _, seen := warnedUnknown.LoadOrStore(name, true); seen {
			continue
		}
		file := ""
		if d.Subject != nil {
			file = d.Subject.String()
		}
		logger.Warn("c3x does not support this function; the value it produces is left unresolved, "+
			"so dependent attributes fall back to catalog defaults and dependent resources may be omitted",
			"function", name, "in", where, "at", file)
	}
}
