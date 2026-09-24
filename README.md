<div align="center">

<img src="https://c3x.dev/android-chrome-512x512.png" width="76" alt="c3x">

# c3x

**Know what your Terraform costs before you apply it.**

Open source cost estimation for Terraform, OpenTofu, Terragrunt and CloudFormation.
No API key, no SaaS account, no telemetry.

[![Release](https://img.shields.io/github/v/release/c3xdev/c3x?color=00ADD8)](https://github.com/c3xdev/c3x/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/c3xdev/c3x/ci.yml?branch=main&label=ci)](https://github.com/c3xdev/c3x/actions)
[![Go](https://img.shields.io/badge/go-1.25%2B-00ADD8.svg)](go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[Documentation](https://c3x.dev/docs) · [Quickstart](https://c3x.dev/docs/quickstart) ·
[CI/CD](https://c3x.dev/docs/ci-cd) · [Resource catalog](https://c3x.dev/resources) ·
[Blog](https://c3x.dev/blog)

</div>

<div align="center">
  <img src="https://raw.githubusercontent.com/c3xdev/c3x/main/docs/demo.gif" width="880"
       alt="c3x estimate printing a per-resource cost breakdown for a two-resource Terraform project, totalling $202.47 per month">
</div>

c3x parses your infrastructure code **statically**. No `terraform init`, no
providers, no cloud credentials, no state access. That makes it safe to run
on untrusted input and fast enough to sit in front of every pull request.

## Install

| Method | Command |
|---|---|
| **Homebrew** | `brew install c3xdev/tap/c3x` |
| **Script** | `curl -fsSL https://c3x.dev/install.sh \| sh` |
| **Docker** | `docker pull ghcr.io/c3xdev/c3x` |
| **Go** | `go install github.com/c3xdev/c3x/cmd/c3x@latest` |

Windows, and anyone who would rather not pipe a script into a shell, can
take the archive straight from the
[releases page](https://github.com/c3xdev/c3x/releases), which carries
macOS, Linux and Windows builds alongside `checksums.txt`. The container
image is on [GHCR](https://github.com/c3xdev/c3x/pkgs/container/c3x).

The script detects your OS and architecture and checks the published
SHA-256, aborting on a mismatch. It skips that check if it cannot fetch
`checksums.txt` or finds no sha256 tool, so verify the download yourself
if you need the guarantee.

Verify the install with `c3x doctor`, which checks the catalog parses,
the pricing API answers, and the cache directory is writable.

## Cost on every pull request

Point c3x at a Terraform plan and it comments the cost change on the PR.
The summary leads with the dollar delta, and the per-resource breakdown
collapses so a large estimate does not flood the thread.

```bash
c3x comment github --path plan.json
```

#### 💰 C3X report

**Monthly cost increased by `$314.82` (+30.3%) 📈**

| Baseline | New | Change |
|---:|---:|---:|
| $1038.32/mo | $1353.14/mo | 🔺 +$314.82 |

<details><summary>Estimate details</summary>

### 🟡 Modified

| Resource | Baseline | Current | Δ |
|---|---:|---:|---:|
| `aws_instance.web` | $285.32 | $565.64 | 🔺 +$280.32 |
| `aws_db_instance.main` | $753 | $787.5 | 🔺 +$34.5 |

</details>

The delta comes from the plan itself. A plan carries both the prior and the
planned state of every resource, so there is no baseline file to manage and
no base-branch checkout. Works on **GitHub, GitLab, Bitbucket and Azure
DevOps**, and auto-detects the repository and PR number in CI.

| Flag | What it does |
|---|---|
| `--comment-tag` | Keeps one comment per environment or directory on the same PR |
| `--recreate` | Posts a fresh comment at the bottom instead of editing in place |
| `--expand` | Posts the breakdown expanded instead of collapsed |
| `--baseline` | Diffs against a saved baseline rather than the plan's prior state |

## In CI

```yaml
- uses: c3xdev/c3x@v0
  with:
    path: .
    budget-delta: "50"   # fail the check if this PR adds more than $50/mo
```

The same gates work from the CLI: `--budget` caps the absolute monthly cost
and `--budget-delta` caps the increase a single change may introduce. Both
exit non-zero, so a pull request that blows the budget fails the build.

See the [CI/CD guide](https://c3x.dev/docs/ci-cd) for GitLab CI, Bitbucket
Pipelines, Azure Pipelines, Atlantis and Spacelift recipes.

## Commands

| Command | What it does |
|---|---|
| `c3x estimate` | Monthly cost of a directory, `.tf` file, or plan JSON |
| `c3x diff` | Cost change against a saved baseline, with a delta gate for CI |
| `c3x top` | Ranks resources by cost. `--format targets` emits `-target=` flags for a teardown |
| `c3x comment` | Posts the estimate to a pull or merge request |
| `c3x recommend` | Suggests savings: right-sizing, cheaper families, unused resources |
| `c3x policy eval` | Evaluates cost policies written in Rego |
| `c3x pricing sync` | Warms the local price cache for fully offline runs |
| `c3x doctor` | Pre-flight checks, usable as a CI gate |
| `c3x supported-resources` | Lists every supported resource kind |

Full flag reference: [c3x.dev/docs/cli](https://c3x.dev/docs/cli).

## OpenTofu

c3x reads OpenTofu projects directly. `.tofu` files are loaded alongside
`.tf`, and a `main.tofu` replaces the `main.tf` of the same name, as
OpenTofu does, so a codebase that ships both is not counted twice.
Provider `for_each` is supported: each instance of a resource is priced
in the region of the provider instance it uses. Plan JSON from
`tofu show -json` works the same as Terraform's.

## Output formats

`text`, `markdown`, `json`, `junit`, `html`, `csv` and `sarif`. JUnit feeds
CI dashboards, SARIF feeds code-scanning surfaces, and JSON is the
machine-readable contract:

```bash
c3x estimate --format json | jq .project_total
```

## Catalog

**1,340 recognized resource kinds** across AWS, Azure and Google Cloud,
served from [pricing.c3x.dev](https://pricing.c3x.dev). Of those, 263 carry
a price and the rest are legitimately free. At estimate time c3x loads the
catalog remote-first: the live `/catalog` bundle, then the on-disk cache,
then an embedded snapshot for fully offline use. Every kind is classified:

| Class | Meaning |
|---|---|
| **live** | Every dimension prices from the upstream API and tracks vendor price changes |
| **static** | Inline-rate fallback for products the upstream does not expose cleanly |
| **free** | Legitimately free, such as public ACM certificates |
| **zero / errored** | A hard CI failure if non-empty |

Regenerate the health report against the live API with
`go run ./cmd/verify_catalog`, which prints the current classification
counts and fails on anything unpriced.

## Performance

| Workload | Time |
|---|---|
| Parse a 500-resource monorepo (vars, locals, count, for_each, modules) | **10ms** (Apple M3 Pro) |
| Cold estimate against pricing.c3x.dev | 3 to 5 seconds (network-bound) |
| Warm estimate (SQLite cache hit) | **35ms** |

From `internal/parser/terraform/bench_test.go`. CI runs the benchmark on
every push, so regressions fail the build.

<details>
<summary><b>Configuration</b></summary>

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

Self-hosting the pricing API? Point `pricing.endpoint` at it and set
`C3X_PRICING_TOKEN` if it runs with `API_KEY` enabled. See
[c3x-pricing-api](https://github.com/c3xdev/c3x-pricing-api).

</details>

<details>
<summary><b>Architecture</b></summary>

```
cmd/c3x/                 # CLI entry (cobra + viper)
cmd/verify_catalog/      # Catalog health harness
internal/
├── domain/              # Core types: Resource, Cost, Estimate, Diff
├── config/              # 5-layer configuration resolution
├── observability/       # slog + OpenTelemetry tracer
├── catalog/             # TOML resource definitions + Registry
├── expr/                # Expression evaluator (expr-lang/expr wrapped)
├── parser/              # IaC parser dispatcher
│   ├── terraform/       # HCL parsing (hashicorp/hcl/v2)
│   └── plan/            # Terraform plan JSON
├── pricing/             # HTTPSource → DiskCache → MemoCache chain
├── calculator/          # Orchestrator
└── render/              # text / markdown / json renderers

resources/               # Embedded TOML catalog snapshot, the
                         # offline fallback shipped in the binary
testdata/corpus/         # Real-world Terraform used by integration tests
```

Module dependencies flow inward toward `domain`. The `depguard` linter
enforces the boundaries in CI, so an import that crosses the wrong line
fails the lint job before review.

</details>

## Stability

c3x is pre-1.0 and versioned honestly. Flags, environment variables and
config keys are not removed without a deprecation cycle, enforced by a
CLI-surface snapshot test that fails CI on a silent removal. See
[COMPATIBILITY.md](COMPATIBILITY.md).

## Contributing

Issues and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md)
covers the layout, the catalog format, and how to add a resource kind.

## License

Apache License 2.0, see [LICENSE](LICENSE).
