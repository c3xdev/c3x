package terraform

import (
	"log/slog"
	"regexp"
	"strings"
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
		if strings.Contains(d.Detail, outsideProjectMsg) {
			if _, seen := warnedUnknown.LoadOrStore("outside:"+d.Detail, true); !seen {
				logger.Warn("refused to read a file outside the project directory; the value is left unresolved",
					"in", where, "detail", d.Detail)
			}
			continue
		}
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
		if fileFunctionNames[name] {
			logger.Warn("file functions are off by default; the value is left unresolved, so dependent "+
				"attributes fall back to catalog defaults and dependent resources may be omitted. "+
				"To evaluate them for your own configuration, pass --allow-file-functions or set "+
				"C3X_ALLOW_FILE_FUNCTIONS=true (not honoured from .c3x.toml, and never with --no-remote-modules)",
				"function", name, "in", where, "at", file)
			continue
		}
		logger.Warn("c3x does not support this function; the value it produces is left unresolved, "+
			"so dependent attributes fall back to catalog defaults and dependent resources may be omitted",
			"function", name, "in", where, "at", file)
	}
}

// fileFunctionNames are the functions registered only when file functions
// are allowed (see evalScope.fileFunctions); a call to one while they are
// off gets a warning that says how to turn them on.
var fileFunctionNames = map[string]bool{
	"file": true, "filebase64": true, "fileexists": true, "fileset": true, "templatefile": true,
	"filemd5": true, "filesha1": true, "filesha256": true, "filesha512": true,
	"filebase64sha256": true, "filebase64sha512": true, "abspath": true,
}
