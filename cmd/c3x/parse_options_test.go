package main

import (
	"testing"

	"github.com/c3xdev/c3x/internal/config"
)

// Every command parses through parserOptions, so untrusted-input mode is
// enforced in one place. Before it existed, only `estimate` honoured
// no_remote_modules; `diff`, `comment` and `recommend` still fetched
// remote modules with it set.
func TestParserOptionsEnforceUntrustedMode(t *testing.T) {
	untrusted := parserOptions(config.Resolved{NoRemoteModules: true, AllowFileFunctions: true}, nil, nil)
	if !untrusted.Offline {
		t.Error("no_remote_modules must disable module fetching")
	}
	if untrusted.AllowFileFunctions {
		t.Error("no_remote_modules must disable file functions even when opted in")
	}
	trusted := parserOptions(config.Resolved{AllowFileFunctions: true}, nil, nil)
	if trusted.Offline || !trusted.AllowFileFunctions {
		t.Errorf("trusted opt-in: got Offline=%v AllowFileFunctions=%v", trusted.Offline, trusted.AllowFileFunctions)
	}
}
