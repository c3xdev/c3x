# Changelog

All notable changes to c3x are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `go install github.com/c3xdev/c3x/cmd/c3x@latest` installed v1.0.2,
  old code from the pre-relaunch lineage, instead of the current
  release. Those versions stayed in the module proxy after their tags
  were removed (proxy content is immutable), and the go command resolves
  `@latest` to the highest release version. `go.mod` now retracts
  v1.0.0 through v1.0.3, published as v1.0.3 because retractions are
  only read from the go.mod of the highest version. v1.0.3 retracts
  itself too, so `@latest` falls back to the 0.x line.

## [0.3.5] - 2026-09-21

### Added

- `c3x top`: ranks the resources in an estimate by monthly cost, highest
  first. `--format targets` emits `-target=<address>` flags so CI can
  feed the costliest resources straight into a destroy, the motivating
  case being scheduled teardown of expensive development environments:
  `tofu destroy $(c3x top --limit 5 --format targets)`. Also renders a
  ranked table (`text`, with each resource's share of the total) and
  `json`. Addresses are canonical Terraform addresses, so resources
  inside modules target correctly; free resources are never listed.
  c3x only prints the list, it never destroys anything. (#64)
- `domain.Reference.TerraformAddress()` builds the canonical Terraform
  address for a reference (`module.frontend.aws_instance.web`), which
  `Label()` does not: the parsers store the module path in `Name` with
  the kind stripped, so the kind has to be spliced back in ahead of the
  final segment. Handles nested modules and `for_each` keys containing
  dots.

## [0.3.4] - 2026-09-18

### Fixed

- Plan-aware diff (`c3x comment` on a plan JSON with no `--baseline`): a
  resource scheduled for pure deletion now renders as a Removed (🔴) row
  and no longer inflates the absolute New / Baseline totals. It was being
  appended to the post-apply set (correct for the single-estimate
  `--show-delta` view) and leaking into the two-estimate diff, where it
  matched itself and was misclassified as Unchanged. The diff's current
  side now parses post-apply-only. The headline dollar change was already
  correct (the cost cancelled in the subtraction). (#62)

## [0.3.3] - 2026-09-17

### Added

- `c3x comment <forge>` now derives the cost delta straight from a
  Terraform plan JSON when no `--baseline` is supplied: a plan already
  carries both sides of every change (`resource_changes[].change.before`
  and the post-apply state), so c3x prices both from the single plan file
  and renders the old/new/change summary with no baseline file and no
  base-branch checkout. Falls back to the absolute estimate for a `.tf`
  directory or a greenfield (all-create) plan; an explicit `--baseline`
  still overrides. Works with exactly what CI already passes
  (`--path=<terraform show -json output>`), no new flags. (#60)

## [0.3.2] - 2026-09-17

### Added

- `c3x comment <forge>` gained `--expand` (all forges): opt out of the
  new collapsible layout and post the full breakdown always-expanded,
  for anyone who preferred the flat comment body. (#55)

### Fixed

- `c3x comment <forge>` now posts the collapsible summary+details
  layout by default: a one-line summary with the per-resource breakdown
  tucked into a `<details>` block, restoring the pre-rewrite comment
  format. With `--baseline`, the summary leads with the dollar change —
  "Monthly cost increased by $144.00 (+16.1%) 📈" plus a
  baseline/new/change table — so financial approvers see the delta
  first; without a baseline it shows the monthly total. Long estimates
  no longer flood the MR/PR discussion thread. The doc comments that
  described this layout are now accurate — the layer was previously
  unimplemented. `c3x estimate --format markdown` is unchanged (flat,
  always-expanded), and `--expand` restores the flat body on the
  comment command. (#55)

## [0.3.1] - 2026-09-16

### Added

- `--pricing-token` flag on `estimate`, `diff`, `recommend`, and
  `pricing sync`, as a CLI alternative to `$C3X_PRICING_TOKEN` /
  `pricing.token` for authenticating to a self-hosted pricing API. The
  flag takes precedence over the env var.

## [0.3.0] - 2026-09-16

### Added

- `c3x comment <forge>` gained `--comment-tag` (all forges): namespaces
  the comment marker so multiple runs on one PR/MR (one per environment
  or Terraform directory in a monorepo) keep independent comments
  instead of overwriting each other. Falls back to `$C3X_COMMENT_TAG`.
  Restores the pre-rewrite per-directory comment workflow. (#55, #56)
- `c3x comment <forge>` gained `--recreate` (all forges): deletes the
  previous c3x comment and posts a fresh one at the bottom of the
  thread instead of updating it in place, for busy PRs where the newest
  estimate reads better at the bottom. (#55, #56)
- Pricing API authentication: set `C3X_PRICING_TOKEN` (or `pricing.token`
  in config) and c3x sends `Authorization: Bearer <token>` on every
  pricing request, including `c3x pricing sync` and `c3x doctor`. This
  lets the CLI talk to a self-hosted `c3x-pricing-api` that has
  `API_KEY` enabled (the server also accepts `X-Api-Key`). Empty by
  default, so the public endpoint is unaffected. (#55, #56)
- `COMPATIBILITY.md` documents the pre-1.0 stability policy: no silent
  removals of flags, env vars, or config keys, a deprecation cycle for
  any removal, and the criteria for 1.0. Enforced by a CLI-surface
  snapshot test (`TestCLISurface`) that fails CI when a flag is removed
  or renamed without going through the policy.

### Fixed

- CLI estimate tests no longer reach the live pricing API; they run
  with the offline stub, removing a class of CI flakes (context
  deadline exceeded) that had nothing to do with the change under test.

## [0.2.1] - 2026-07-28

### Added

- `--show-delta` flag for `c3x estimate`: shows only resources with
  plan actions (create/update/delete) annotated with `+`/`~`/`-`
  markers, summarizes unchanged resources in a footer, and displays a
  cost delta line. Only meaningful for plan JSON input; warns and falls
  back to the standard view when used with `.tf` files.
- `optional()` type-constraint defaults are now resolved for both root
  and child module variables through Terraform's own `typeexpr` type
  system. Explicit `optional(type, default)` defaults are applied, bare
  `optional(type)` attributes materialize as null, and `map`/`list` of
  `object(...)` wrappers are handled — matching Terraform's runtime
  behavior instead of bespoke, partial extraction.

### Fixed

- Plan parser: `provider_config.expressions` no longer panics on
  array-valued expressions (e.g. azurerm `features` block in
  Terraform 4.x). Uses `json.RawMessage` with graceful skip.
- Registry modules that assign an attribute through an `optional()`
  object input now reflect the real value instead of silently falling
  back to the catalogue default. Previously an input typed
  `map(object({ instance_class = optional(string) }))` left the
  attribute absent, so an expression like
  `try(coalesce(each.value.instance_class, var.cluster_instance_class),
  null)` errored on the missing attribute, `try()` swallowed it to null,
  and the resource priced at the default class — the delta column missed
  the change entirely (e.g. terraform-aws-modules/rds-aurora). (#53)

## [0.1.0] - 2026-06-22

First public release.

### Inputs

- Terraform (`.tf`) parsing — variables, locals, `count`, `for_each`,
  and module resolution (local and registry/git sources).
- Terraform plan JSON.
- Terragrunt (`terragrunt.hcl`) — `terraform.source`, `inputs`,
  `include` walking, and locals resolution.
- CloudFormation templates (YAML and JSON).

### Pricing

- Live pricing from `pricing.c3x.dev`, loaded remote-first: the
  `/catalog` knowledge base, then an on-disk SQLite cache, then an
  embedded snapshot for fully-offline use (`--offline`).
- Declarative TOML catalog with an `expr`-based dimension evaluator.
- `verify_catalog` health harness classifying every kind as
  live / static / free / zero / errored.

### Output

- Per-resource cost breakdowns with monthly totals and multi-currency
  display.
- `c3x diff` against a saved baseline, with budget gates (`--budget`).
- Recommendations: gp2→gp3, EBS right-sizing, non-prod Multi-AZ RDS
  downgrade, idle EIP audit, Azure burstable swap, GCP
  pd-standard→pd-balanced, and cross-resource NAT consolidation with
  net-savings math.
- `--what-if` overrides and usage files for usage-driven dimensions.
- Formats: text, markdown, JSON, JUnit, HTML, CSV, SARIF.
- PR comments for GitHub, GitLab, Bitbucket, and Azure DevOps.
- Rego policy gates (`c3x policy eval`).

### Tooling

- 5-layer configuration (defaults → user → project → env → flags).
- `c3x doctor` pre-flight checks; `c3x pricing` cache inspection.
- Single static binary; Homebrew, Docker (GHCR), and install-script
  distribution.

[0.1.0]: https://github.com/c3xdev/c3x/releases/tag/v0.1.0
