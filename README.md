# c3x

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)](go.mod)
[![Status](https://img.shields.io/badge/status-pre--release-orange.svg)]()

Cloud cost estimation for Terraform and CloudFormation. Open source,
no API key, no SaaS, no telemetry phone-home. Pulls live prices from
[pricing.c3x.dev](https://pricing.c3x.dev) and caches them locally.

```
$ c3x estimate
── c3x estimate · USD ─────────────────────────────────────────

  aws_instance.web
    Instance usage (Linux/UNIX, on-demand)
      730 hours × $0.192 = $140.16/mo
    aws_instance.web subtotal: $140.16/mo

  aws_db_instance.primary
    Database instance
      730 hours × $0.068 = $49.64/mo
    Storage
      50 GB-month × $0.115 = $5.75/mo
    aws_db_instance.primary subtotal: $55.39/mo

  ──────────────────────────────────────────────────────────
  PROJECT TOTAL: $195.55/mo
```

## Install

```bash
# Homebrew (macOS)
brew install c3xdev/tap/c3x

# Install script — detects your OS/arch, verifies the checksum
curl -fsSL https://c3x.dev/install.sh | sh

# Docker
docker pull ghcr.io/c3xdev/c3x

# From source
go install github.com/c3xdev/c3x/cmd/c3x@latest
```

Pre-built binaries and checksums for every release are on the
[releases page](https://github.com/c3xdev/c3x/releases).

## Usage

```bash
# Cost estimate of the current directory's Terraform config.
c3x estimate

# Run against a specific path; emit JSON for machine consumption.
c3x estimate --path infra/ --format json | jq .project_total

# Markdown for a PR comment.
c3x estimate --format markdown > pr-comment.md

# Override a variable.
c3x estimate --var 'env="prod"' --var-file=overrides.tfvars

# Run offline (no calls to pricing.c3x.dev; uses the offline stub).
c3x estimate --offline

# Rank resources by monthly cost.
c3x top --limit 10

# Emit the costliest resources as destroy targets, e.g. to tear down an
# expensive dev environment on a schedule. c3x only prints the list;
# review it before destroying anything.
tofu destroy $(c3x top --limit 5 --format targets)

# Inspect the on-disk price cache.
c3x pricing where
c3x pricing stats
c3x pricing clear
```

## Configuration

c3x reads from five layers, in increasing priority:

1. Built-in defaults
2. User config: `~/.config/c3x/config.toml`
3. Project config: `./.c3x.toml`
4. Environment variables prefixed `C3X_`
5. CLI flags

Project config example (`.c3x.toml`):

```toml
region = "us-east-1"
format = "markdown"
budget = 1000.0

[pricing]
endpoint = "https://pricing.c3x.dev/graphql"
```

## Architecture

```
cmd/c3x/                 # CLI entry (cobra + viper)
cmd/verify_catalog/      # Catalog health harness
internal/
├── domain/              # Core types — Resource, Cost, Estimate, Diff
├── config/              # 5-layer configuration resolution
├── observability/       # slog + OpenTelemetry tracer
├── catalog/             # Embedded TOML resource definitions + Registry
├── expr/                # Expression evaluator (expr-lang/expr wrapped)
├── parser/              # IaC parser dispatcher
│   ├── terraform/       # HCL parsing (hashicorp/hcl/v2)
│   └── plan/            # Terraform plan JSON
├── pricing/             # HTTPSource → DiskCache → MemoCache chain
├── calculator/          # Orchestrator
└── render/              # text / markdown / json renderers

resources/               # Embedded TOML catalog snapshot — the
                         # offline fallback shipped inside the binary
testdata/corpus/         # Real-world Terraform configurations used
                         # by integration tests
```

Module dependencies flow inward toward `domain`. The `depguard`
linter enforces the dependency rules in CI; an import that crosses
the wrong boundary fails the lint job before review.

## Performance

| Workload | Time |
|---|---|
| Parse a 500-resource monorepo (vars + locals + count + for_each + modules) | **10ms** (Apple M3 Pro) |
| Cold estimate against pricing.c3x.dev | 3-5 seconds (network-bound) |
| Warm estimate (SQLite cache hit) | **35ms** |

Numbers from `internal/parser/terraform/bench_test.go`; CI runs the
benchmark on every push so regressions fail the build.

## Catalog

The catalog is the knowledge base served by `pricing.c3x.dev`. At
estimate time c3x loads it remote-first: the live `/catalog` bundle,
then the on-disk cache, then an embedded snapshot for fully-offline
use. Each kind is classified:

- **live** — every dimension prices via the upstream API and tracks
  vendor price changes automatically.
- **static** — inline-rate fallbacks for products the upstream
  doesn't expose cleanly; these don't track upstream changes.
- **free** — legitimately-free resources (e.g. public ACM certs, ALB
  target groups that bill via their parent).
- **zero / errored** — a hard CI failure if non-empty.

Run the verifier against the live API to regenerate the health
report (see [CATALOG_STATUS.md](CATALOG_STATUS.md) for current
counts):

```bash
go run ./cmd/verify_catalog
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
