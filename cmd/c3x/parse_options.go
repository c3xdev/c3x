package main

import (
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/parser"
)

// parserOptions builds the options every command parses with, so the
// untrusted-input guarantees apply uniformly. They used to be assembled at
// each call site, and only `estimate` honoured no_remote_modules: `diff`,
// the plan-aware path behind `comment`, and `recommend` would still fetch
// remote modules with it set.
func parserOptions(resolved config.Resolved, varFiles []string, vars map[string]string) parser.Options {
	return parser.Options{
		VarFiles: varFiles,
		Vars:     vars,
		// Remote modules are not fetched when pricing is offline or in
		// untrusted-input mode (no_remote_modules). The pricing chain honours
		// resolved.Offline independently.
		Offline: resolved.Offline || resolved.NoRemoteModules,
		// Already forced off by config in untrusted-input mode; repeated
		// here so the guarantee does not rest on one place.
		AllowFileFunctions: resolved.AllowFileFunctions && !resolved.NoRemoteModules,
		// Confines local module sources to the scanned directory.
		Untrusted: resolved.NoRemoteModules,
	}
}
