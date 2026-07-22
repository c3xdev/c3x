# Changelog

All notable changes to c3x are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `--show-delta` flag for `c3x estimate`: shows only resources with
  plan actions (create/update/delete) annotated with `+`/`~`/`-`
  markers, summarizes unchanged resources in a footer, and displays a
  cost delta line. Only meaningful for plan JSON input; warns and falls
  back to the standard view when used with `.tf` files.
- `optional()` type-constraint defaults are now resolved for both root
  and child module variables, matching Terraform's runtime behavior.
  Previously, attributes using `optional(type, default)` could resolve
  to nil or a source-range string, causing type mismatches in catalog
  expressions.

### Fixed

- Plan parser: `provider_config.expressions` no longer panics on
  array-valued expressions (e.g. azurerm `features` block in
  Terraform 4.x). Uses `json.RawMessage` with graceful skip.
- `optional_defaults.go`: no longer panics on zero-arg `object()` type
  constraints in HCL.
- `optional(object({...}))` without an explicit default no longer
  synthesizes a phantom object from child defaults — matching
  Terraform's behavior of leaving the attribute null.

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
