package main

// CLI surface guard. This snapshots every command's flags into a
// checked-in golden file (testdata/cli_surface.golden). Adding a flag
// is a visible one-line golden diff; REMOVING or RENAMING a flag is a
// deletion in that diff, which fails CI and forces the change through
// the compatibility policy in COMPATIBILITY.md (deprecate with a
// warning for a release, don't silently drop).
//
// This is the mechanical backstop for the promise made in issue #55:
// no silent removals of the public CLI surface.
//
// To intentionally change the surface, regenerate the golden:
//
//	UPDATE_CLI_SURFACE=1 go test ./cmd/c3x/ -run TestCLISurface
//
// and review the resulting diff.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const cliSurfaceGolden = "testdata/cli_surface.golden"

func TestCLISurface(t *testing.T) {
	got := renderCLISurface(newRootCmd())

	if os.Getenv("UPDATE_CLI_SURFACE") == "1" {
		if err := os.MkdirAll(filepath.Dir(cliSurfaceGolden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cliSurfaceGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", cliSurfaceGolden)
		return
	}

	wantBytes, err := os.ReadFile(cliSurfaceGolden)
	if err != nil {
		t.Fatalf("reading golden: %v (run with UPDATE_CLI_SURFACE=1 to create it)", err)
	}
	if got != string(wantBytes) {
		t.Errorf(`CLI surface changed.

If you ADDED a command or flag, regenerate the golden:
    UPDATE_CLI_SURFACE=1 go test ./cmd/c3x/ -run TestCLISurface

If you REMOVED or RENAMED a flag, that is a breaking change to the
public CLI surface. Per COMPATIBILITY.md, deprecate it instead: keep
the old flag working with a deprecation warning for at least one minor
release before removal. Then regenerate the golden.

--- want (golden)
%s
--- got (current)
%s`, string(wantBytes), got)
	}
}

// renderCLISurface walks the command tree and emits one deterministic
// line per command: "<full path>: --flag --flag ...". Only local flags
// are listed per command (persistent flags appear once on the command
// that defines them); the root's persistent --verbose thus shows on the
// "c3x" line and is inherited everywhere.
func renderCLISurface(root *cobra.Command) string {
	var lines []string
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		var flags []string
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, "--"+f.Name)
		})
		sort.Strings(flags)
		lines = append(lines, cmd.CommandPath()+": "+strings.Join(flags, " "))
		children := cmd.Commands()
		sort.Slice(children, func(i, j int) bool {
			return children[i].Name() < children[j].Name()
		})
		for _, c := range children {
			if c.Name() == "help" || c.Name() == "completion" {
				continue // cobra built-ins, not part of our surface
			}
			walk(c)
		}
	}
	walk(root)
	return strings.Join(lines, "\n") + "\n"
}
