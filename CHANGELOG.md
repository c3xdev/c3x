# Changelog

All notable changes to C3X will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-10

### Added

- **Cost estimation engine** with support for 1,100+ Terraform resources across AWS, Azure, and Google Cloud.
- **Independent pricing API** that scrapes directly from cloud provider APIs (AWS Bulk Pricing, Azure Retail Prices, GCP Cloud Billing) — no third-party dependencies.
- **Cost optimization recommendations** (`c3x recommend`) — suggests newer instance generations, better storage types (gp2 to gp3), and architectural improvements (VPC endpoints for NAT Gateway).
- **Budget guardrails** (`--budget`, `--budget-increase`) — enforce cost limits in CI/CD pipelines. Exit code 1 when budget is exceeded. No paid subscription required.
- **What-if cost scenarios** (`--what-if`) — explore cost impact of resource changes without modifying Terraform code.
- **Fully offline mode** (`c3x pricing sync` + `--offline`) — download cloud pricing data to a local SQLite database and estimate costs without any network calls.
- **Self-hosted pricing** — run your own pricing API with `C3X_SELF_HOSTED=true`. No API key required.
- **CLI commands**:
  - `c3x estimate` — Generate cost estimates from Terraform, Terragrunt, or CloudFormation.
  - `c3x diff` — Compare cost changes between two states.
  - `c3x recommend` — Suggest cost optimizations for estimated resources.
  - `c3x pricing sync` — Download pricing data for offline use.
  - `c3x pricing status` — Show local pricing database status.
  - `c3x report` — Format estimates as table, JSON, HTML, or Markdown.
  - `c3x config` — Manage C3X configuration.
  - `c3x comment` — Post cost estimates as PR comments (GitHub, GitLab, Bitbucket, Azure Repos).
  - `c3x auth` — Authenticate with C3X Cloud.
  - `c3x upload` — Upload estimates to C3X Cloud dashboard.
- **Output formats**: Table, JSON, HTML, Markdown, Slack message, and diff views.
- **Usage-based estimation** via `c3x-usage.yml` files for resources with variable pricing (e.g., data transfer, API requests, storage).
- **CI/CD integration** with GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure Pipelines, Atlantis, and Spacelift.
- **Terragrunt support** with parallel module evaluation and dependency graph resolution.
- **CloudFormation support** for AWS CloudFormation templates.
- **Policy engine** for setting cost guardrails and governance rules.
- **Docker images** for containerized CI/CD workflows.
- **Cross-platform binaries** for Linux, macOS, and Windows (amd64/arm64).
