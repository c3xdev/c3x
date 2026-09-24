package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/config"
)

// allow_file_functions decides whether a parse can read files, so where it
// may come from is the security boundary. These pin every source.

func resolveAllow(t *testing.T, projectTOML, userTOML string, flags map[string]any) bool {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if userTOML != "" {
		if err := os.MkdirAll(filepath.Join(xdg, "c3x"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xdg, "c3x", "config.toml"), []byte(userTOML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	project := t.TempDir()
	if projectTOML != "" {
		if err := os.WriteFile(filepath.Join(project, ".c3x.toml"), []byte(projectTOML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := config.Resolve(project, flags)
	if err != nil {
		t.Fatal(err)
	}
	return r.AllowFileFunctions
}

func TestAllowFileFunctionsOffByDefault(t *testing.T) {
	if resolveAllow(t, "", "", nil) {
		t.Error("file functions must be off by default")
	}
}

// The project config lives in the scanned directory, which a pull request
// can edit. A PR adding `allow_file_functions = true` must not enable it.
func TestAllowFileFunctionsIgnoredInProjectConfig(t *testing.T) {
	if resolveAllow(t, "allow_file_functions = true\n", "", nil) {
		t.Error("allow_file_functions must not be honoured from the project's .c3x.toml")
	}
}

func TestAllowFileFunctionsFromRunnerControlledSources(t *testing.T) {
	if !resolveAllow(t, "", "allow_file_functions = true\n", nil) {
		t.Error("user config should enable file functions")
	}
	if !resolveAllow(t, "", "", map[string]any{"allow_file_functions": true}) {
		t.Error("the flag should enable file functions")
	}
	t.Setenv("C3X_ALLOW_FILE_FUNCTIONS", "true")
	if !resolveAllow(t, "", "", nil) {
		t.Error("C3X_ALLOW_FILE_FUNCTIONS should enable file functions")
	}
}

// Untrusted-input mode wins over every opt-in, however it is set.
func TestNoRemoteModulesForcesFileFunctionsOff(t *testing.T) {
	t.Setenv("C3X_ALLOW_FILE_FUNCTIONS", "true")
	flags := map[string]any{"allow_file_functions": true, "no_remote_modules": true}
	if resolveAllow(t, "", "allow_file_functions = true\n", flags) {
		t.Error("no_remote_modules must force file functions off")
	}
	// Set through the project config rather than a flag, it still wins.
	if resolveAllow(t, "no_remote_modules = true\n", "", map[string]any{"allow_file_functions": true}) {
		t.Error("no_remote_modules from any source must force file functions off")
	}
}
