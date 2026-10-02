package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/config"
)

// resolveIn is resolveWith that also returns the project directory, so a
// test can create files next to .c3x.toml.
func resolveIn(t *testing.T, projectTOML string, setup func(dir string), flags map[string]any) (config.Resolved, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	project := t.TempDir()
	if projectTOML != "" {
		if err := os.WriteFile(filepath.Join(project, ".c3x.toml"), []byte(projectTOML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if setup != nil {
		setup(project)
	}
	r, err := config.Resolve(project, flags)
	if err != nil {
		t.Fatal(err)
	}
	return r, project
}

func TestSyncedUsagePathUnsetByDefault(t *testing.T) {
	r, _ := resolveIn(t, "", nil, nil)
	if r.SyncedUsagePath != "" {
		t.Errorf("synced_usage_path = %q, want empty when there is no snapshot", r.SyncedUsagePath)
	}
}

func TestSyncedUsagePathDefaultsToSnapshotInProject(t *testing.T) {
	r, project := resolveIn(t, "", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, config.DefaultSyncedUsageFile), []byte("version: \"0.1\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}, nil)
	if want := filepath.Join(project, config.DefaultSyncedUsageFile); r.SyncedUsagePath != want {
		t.Errorf("synced_usage_path = %q, want %q", r.SyncedUsagePath, want)
	}
}

func TestSyncedUsagePathDefaultSkipsSymlinks(t *testing.T) {
	r, _ := resolveIn(t, "", func(dir string) {
		if err := os.Symlink("/etc/passwd", filepath.Join(dir, config.DefaultSyncedUsageFile)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}, nil)
	if r.SyncedUsagePath != "" {
		t.Errorf("synced_usage_path = %q; a symlinked snapshot must not be followed", r.SyncedUsagePath)
	}
}

func TestSyncedUsagePathIsRelativeToProject(t *testing.T) {
	r, project := resolveIn(t, `synced_usage_path = "usage/synced.yml"`+"\n", nil, nil)
	if want := filepath.Join(project, "usage", "synced.yml"); r.SyncedUsagePath != want {
		t.Errorf("synced_usage_path = %q, want %q", r.SyncedUsagePath, want)
	}
}

func TestSyncedUsagePathFlagAndEnvWinOverProject(t *testing.T) {
	r, _ := resolveIn(t, `synced_usage_path = "from-project.yml"`+"\n", nil,
		map[string]any{"synced_usage_path": "from-flag.yml"})
	if r.SyncedUsagePath != "from-flag.yml" {
		t.Errorf("synced_usage_path = %q, want the flag's value", r.SyncedUsagePath)
	}

	t.Setenv("C3X_SYNCED_USAGE_PATH", "from-env.yml")
	r, _ = resolveIn(t, `synced_usage_path = "from-project.yml"`+"\n", nil, nil)
	if r.SyncedUsagePath != "from-env.yml" {
		t.Errorf("synced_usage_path = %q, want the environment's value", r.SyncedUsagePath)
	}
}

func TestUntrustedModeKeepsSyncedUsagePathInsideProject(t *testing.T) {
	r, project := resolveIn(t, "synced_usage_path = \"synced.yml\"\n", nil, map[string]any{"no_remote_modules": true})
	if want := filepath.Join(project, "synced.yml"); r.SyncedUsagePath != want {
		t.Errorf("an in-project synced_usage_path = %q, want %q kept", r.SyncedUsagePath, want)
	}

	r, _ = resolveIn(t, "synced_usage_path = \"/etc/passwd\"\n", nil, map[string]any{"no_remote_modules": true})
	if r.SyncedUsagePath != "" {
		t.Errorf("synced_usage_path = %q; a path outside the project must be dropped when untrusted", r.SyncedUsagePath)
	}
}
