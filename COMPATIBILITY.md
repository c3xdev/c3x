# Compatibility policy

c3x is pre-1.0 and versioned honestly under SemVer 0.x: the public
surface is still stabilizing. "Pre-1.0" does not mean "unstable to
build on." It means changes are deliberate, announced, and never
silent. This document is the commitment and how it is enforced.

## The public surface

These are treated as public contract:

- CLI commands and their flags
- Environment variables c3x reads (`C3X_*`, forge tokens, etc.)
- Config keys in `.c3x.toml` / user config
- The exit-code contract (`--budget` / `--budget-delta` fail with a
  non-zero exit)

Output formatting (table layout, wording) and internal Go packages are
not public contract and may change between minor releases.

## No silent removals

A flag, env var, or config key in a released version is never deleted
without a deprecation cycle:

1. Keep it working. Mark the flag deprecated with a warning:

   ```go
   cmd.Flags().MarkDeprecated("old-flag", "use --new-flag instead")
   ```

   `MarkDeprecated` prints the warning when the flag is used and hides
   it from help, while the flag keeps functioning. For env vars and
   config keys, keep reading the old key and log a warning when it is
   set.

2. Record it under `### Deprecated` (or `### Removed`) in
   `CHANGELOG.md`, naming the replacement.

3. Leave the deprecation in place for at least one minor release before
   removing it.

Renames follow the same path: add the new name, deprecate the old one,
remove the old one no sooner than the next minor release.

## How this is enforced

`cmd/c3x/cli_surface_test.go` snapshots every command's flags into
`cmd/c3x/testdata/cli_surface.golden`. CI fails if the live surface
diverges from the golden. Adding a flag is a one-line golden addition;
removing or renaming one is a deletion in that diff, which fails the
build and forces the change through the steps above. Regenerate the
golden intentionally with:

```bash
UPDATE_CLI_SURFACE=1 go test ./cmd/c3x/ -run TestCLISurface
```

The golden diff is also the reviewer's signal that a PR touches the
public surface and therefore needs a CHANGELOG entry. Env vars and
config keys are not enumerable the same way; they are held to this
policy by review.

## Path to 1.0

1.0 ships when:

- the CLI surface (commands, flags, env vars, config keys) is frozen, and
- a migration guide covers every breaking change from the current line.

Until then, 0.x is stable to build on and versioned honestly, under the
policy above.
