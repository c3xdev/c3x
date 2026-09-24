# Changelog

All notable changes to c3x are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.8] - 2026-09-24

### Added

- OpenTofu support. `.tofu` files are read alongside `.tf`, and a
  `.tofu` file shadows the `.tf` file of the same name, as OpenTofu
  does, so a project carrying both is not double-counted; a single
  `.tofu` file can be passed to `--path`. OpenTofu 1.9 provider
  `for_each` is supported: `provider = aws.by_region[each.key]` prices
  each instance in its own provider instance's region. Verified against
  OpenTofu 1.12.6.
- Terraform and OpenTofu functions the evaluator lacked: `one`, `sum`,
  `alltrue`, `anytrue`, `startswith`, `endswith`, `strcontains`,
  `templatestring`, `chunklist`, `sensitive`, `nonsensitive`,
  `issensitive`, `base64encode`/`decode`, `base64gzip`, `urlencode`,
  `yamldecode`/`encode`, `md5`, `sha1`, `sha256`, `sha512`,
  `base64sha256`/`512`, `cidrhost`, `cidrnetmask`, `cidrsubnet`,
  `cidrsubnets`, and OpenTofu's `cidrcontains`, `urldecode` and
  `base64gunzip`. An unknown function was silent where it hurt: in an
  attribute it priced the catalog default (`templatestring` in an
  `instance_type` came out as a t3.micro), and in a local it dropped
  every resource reading it (a `cidrsubnet` in a `for_each` local
  removed three NAT gateways from an estimate).
- A warning naming the function and location when a configuration calls
  one c3x still does not implement (the filesystem functions such as
  `file` and `templatefile`, `timestamp`, `uuid`), instead of a silently
  wrong estimate.

### Fixed

- Resources on an aliased provider are priced in that provider's
  region. Every resource used to take the region of the first provider
  block, so a default `us-east-1` provider plus an `aws.eu` alias priced
  both halves at us-east-1 rates. Module `providers = { aws = aws.eu }`
  mappings are honoured as well.
- Plan JSON: each resource is priced in the region the plan resolved
  for it (the `region` attribute AWS provider v6 and Google resources
  carry), falling back to the configuration default when absent.
- Plan JSON: `count` and `for_each` instances keep their keys, so
  `web["a"]` and `web["b"]` are no longer both named `web`. Diffs pair
  resources by name, so every instance was compared against the first
  one: a PR comment for a plan downsizing `web["b"]` listed both
  instances as unchanged while its total showed the saving. Names now
  also match what the `.tf` parser produces.
  The same loss affected `c3x top --format targets` since it shipped in
  v0.3.5: from a plan JSON, the most expensive instance of a `for_each`
  resource was emitted as `-target=aws_instance.web`, which covers every
  instance, instead of `-target=aws_instance.web["b"]`. Reading a `.tf`
  directory was not affected.

### Changed

- Resource names from plan JSON now include the instance key. A
  baseline saved from a plan by an earlier version will show those
  resources as removed and re-added on the first diff against a new
  one; re-save the baseline to clear it.

## [0.3.7] - 2026-09-23

### Fixed

- Aurora I/O-Optimized clusters now price their instances at the
  I/O-Optimized rate. `storage_type` is declared on `aws_rds_cluster`
  while the hours are billed on `aws_rds_cluster_instance`, so the
  parser copies declared attributes from a parent resource onto the
  child billed for them, before pricing. Flipping a cluster to
  `aurora-iopt1` moves the instance from $0.26 to $0.338 per hour and
  storage from $0.10 to $0.225 per GB-month. (#68)

### Changed

- `linked()` and the cross-resource expression context added in 0.3.6
  are removed. The catalog is fetched remote-first, so an expression
  calling a function an older engine lacks breaks that client, and
  gating it behind a schema bump is worse: a client that rejects the
  bundle falls back to its own embedded catalog, which for <= 0.3.5
  still overcharges Aurora. The capability belongs in the client, where
  an older release simply does not benefit instead of breaking. Both
  catalog copies are byte identical again.

## [0.3.6] - 2026-09-22

### Fixed

- Aurora clusters were priced at the I/O-Optimized rate whether or not
  they used it. Aurora publishes two instance SKUs per class that share
  `instanceType`, `databaseEngine` and `deploymentOption`, which was all
  the mapping filtered on, so both matched and the max-non-zero picker
  took the dearer one. A `db.r6g.large` priced at $246.74/mo instead of
  $189.80, a roughly 30% overcharge on Aurora Standard, and switching to
  `aurora-iopt1` appeared to cost nothing because the premium was
  already being charged. The instance now discriminates on the `storage`
  attribute, and cluster storage follows `storage_type`. (#68)
- Aurora I/O was priced at zero everywhere except us-east-1: the
  `usagetype` filter pinned `Aurora:StorageIOUsage`, but upstream that
  value carries a region prefix (`EU-Aurora:StorageIOUsage`), so it
  matched nothing. The pin is gone; `group` already narrows the meter.
- Aurora I/O is no longer billed on clusters using `aurora-iopt1`, where
  it is included in the storage rate.

### Added

- `linked(kind, join_attr, wanted_attr)` in catalog expressions: reads an
  attribute from a related resource, joined on an attribute the two
  share. It exists because some costs are decided elsewhere than where
  they are billed, as with Aurora, where `storage_type` is declared on
  `aws_rds_cluster` but the hours are billed on
  `aws_rds_cluster_instance`. Returns "" when nothing matches, so
  expressions fall back with `default()` instead of failing on
  unresolved HCL references. Used by the embedded catalog; the copy
  served from pricing.c3x.dev keeps the compatible form until a catalog
  schema bump lets older clients fall back safely.

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
