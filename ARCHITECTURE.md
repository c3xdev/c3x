# c3x architecture

A ten-minute tour of how c3x turns an IaC config into a monthly
cost estimate. Targets contributors who need to land changes without
spelunking the code first.

## Pipeline at a glance

```
   Terraform / CFN / plan JSON
              │
              ▼
   ┌──────────────────────┐
   │   internal/parser    │   format-detect + dispatch to
   │  ┌────────────────┐  │   terraform / cloudformation /
   │  │  terraform/    │  │   terragrunt / plan
   │  │  cloudformation│  │
   │  │  terragrunt/   │  │
   │  │  plan/         │  │
   │  └────────────────┘  │
   └──────────────────────┘
              │
              ▼
       []domain.Resource
              │
              ▼
   ┌──────────────────────┐
   │  internal/usage      │   usage YAML overlays
   │  internal/whatif     │   `kind.name.attr=value` overrides
   └──────────────────────┘
              │
              ▼
   ┌──────────────────────┐         ┌──────────────────────┐
   │  internal/calculator │ ──────▶ │   internal/catalog   │
   │      engine          │         │  declarative TOMLs   │
   │                      │         └──────────────────────┘
   │                      │ ──────▶ ┌──────────────────────┐
   │                      │         │   internal/pricing   │
   │                      │         │  memo → sqlite → http│
   │                      │         └──────────────────────┘
   └──────────────────────┘
              │
              ▼
        domain.Estimate
              │
              ▼
   ┌──────────────────────┐
   │   internal/render    │   text / markdown / json / junit
   └──────────────────────┘
```

## Package responsibilities

| Package | Concern | What it owns |
|---|---|---|
| `internal/domain` | Vocabulary | `Resource`, `Reference`, `Cost`, `LineItem`, `Estimate`, `Diff`, `Currency`. No logic, only types. Every other package depends on this. |
| `internal/parser` | Input → resources | Sniffs format from path/extension, dispatches to a backend. Backends produce `[]domain.Resource`. |
| `internal/parser/terraform` | HCL pipeline | Full `vars → tfvars → CLI vars → locals (fixed-point) → data placeholders → resources → modules` resolution using `hashicorp/hcl/v2` + `zclconf/go-cty/cty`. |
| `internal/parser/cloudformation` | CFN → resources | YAML/JSON with short-form intrinsic-tag rewriting (`!Ref`, `!Sub`, `!Join`, `!GetAtt`, `!FindInMap`). |
| `internal/parser/plan` | Terraform plan and state JSON | Reads `terraform show -json` output: a plan (post-apply values) or a state (what exists, with real identifiers). |
| `internal/parser/terragrunt` | Terragrunt → terraform | `terraform.source`, `inputs`, `locals`, `find_in_parent_folders()` resolution; delegates to terraform parser. |
| `internal/catalog` | Resource definitions | Embedded TOMLs at `resources/<provider>/<kind>.toml` parsed once into a `*Registry`. Each definition has mappings, dimensions, and a snapshot fixture. |
| `internal/expr` | Expression DSL | Wraps `expr-lang/expr` with the c3x stdlib (`default`, `pick`, `monthly_hours`, `price`, `replace`, …). Compiled programs are cached in `internal/calculator/programCache`. |
| `internal/pricing` | Per-unit rates | `Source` interface with implementations `HTTPSource` → `DiskCache` (SQLite, 7-day TTL) → `MemoCache` (in-process). Production chain: `MemoCache(DiskCache(HTTPSource))`. |
| `internal/calculator` | Estimate engine | Walks resources × catalog definitions × pricing source, emitting `domain.Estimate`. Owns the program cache. |
| `internal/recommend` | Cost optimizations | Per-resource and tree (cross-resource) rules. Engine re-estimates proposed alternatives and reports savings. |
| `internal/render` | Estimates → strings | One renderer per output format. Pure functions of `domain.Estimate` + `Format`. |
| `internal/usage` | Usage YAML | Loads `c3x-usage.yml` and merges its attributes into parsed resources. |
| `internal/usagesync` | Usage from cloud metrics | `c3x usage sync`: finds the resources to look up (real names from Terraform state), runs the lookups with per-resource failure, and writes the generated `c3x-usage.synced.yml`. Provider-neutral behind a `Source` interface; never touches prices. Only `cmd/c3x` imports it. |
| `internal/usagesync/aws` | AWS metric source | The only package that imports the AWS SDK (`config`, `cloudwatch`). Read-only CloudWatch calls. |
| `internal/whatif` | CLI overrides | Parses `--what-if kind.name.attr=value` syntax. |
| `internal/comment` | Forge integrations | GitHub PR comment poster with marker-based update-in-place. Interface seam for GitLab/Bitbucket/Azure DevOps. |
| `internal/config` | User config | 5-layer resolution: defaults → file → env → CLI → run-time. |
| `internal/observability` | Logging + tracing | `slog` wrapper + OpenTelemetry API-only tracer (no SDK dep → zero-cost by default). |

## How the catalog works

Every supported resource type lives in `resources/<provider>/<kind>.toml`.
There is **no Go code per resource**. A TOML looks like:

```toml
kind         = "aws_dax_cluster"
display_name = "AWS DAX Cluster"
provider     = "aws"

[mappings.node]
service        = "AmazonDAX"
product_family = "Amazon DynamoDB Accelerator (DAX)"
attribute_filters = [
  { key = "instanceType", expr = 'default(node_type, "dax.t3.small")' },
]

[[dimensions]]
id       = "node_hours"
label    = "DAX nodes"
unit     = "node-hours"
quantity = "default(replication_factor, 1) * monthly_hours()"
rate     = 'price("node")'

[fixture]
attributes = { node_type = "dax.t3.small", replication_factor = 3 }
expected_monthly_cost = 87.6
exact = true
```

- **Mappings** are upstream-pricing-API filter recipes. Each has a
  service + productFamily + attribute filters. Catalog authors
  reference them from dimensions via `price("name")`.
- **Dimensions** are billable rows. `quantity` and `rate` are
  expression-language strings evaluated against the parsed
  resource's attributes plus the c3x stdlib.
- **Fixture** is the canonical test resource: attributes the
  verifier uses to exercise the kind end-to-end, plus the expected
  monthly cost. Snapshot-based; the verifier reports DRIFT when an
  upstream rate change moves the computed cost outside the
  tolerance band.

Adding a new resource is one file. Validation enforces:
- Unique kind across the whole tree
- Unique dimension `id` within a kind (collision → program-cache
  bug, caught at load)
- Every `price("…")` reference resolves to a declared mapping

## Pricing path

The calculator never talks HTTP. It calls `pricing.Source.Lookup`
which goes through the cache stack:

```
   Lookup(Query)
        │
        ▼
   MemoCache (in-process map, never expires per process)
        │ miss
        ▼
   DiskCache (SQLite, 7-day TTL)
        │ miss / expired
        ▼
   HTTPSource (pricing.c3x.dev/graphql, 30s timeout, exponential backoff retries)
        │
        ▼
   pickNonZeroPrice → decimal.Decimal
```

The `pickNonZeroPrice` step is conservative: when a product returns
several price tiers, we pick the **largest non-zero** value. This
protects against silently understating cost by selecting a discount
tier the user hasn't actually committed to.

## Recommendation engine

Two contracts:

- `Rule.Propose(r) []Proposal` — per-resource. Engine re-estimates
  the proposed alternative and reports the delta if positive.
- `TreeRule.ProposeTree(rs) []TreeProposal` — cross-resource.
  Engine re-estimates the **whole modified set** and reports net
  delta. Used for NAT consolidation (drop NAT charges + add
  inter-AZ transfer cost), idle ALB detection, fleet-level
  Committed Use Discounts.

Both surfaces are free; no separate paid tier.

## Concurrency

- The catalog `Registry` is read-only after `Load()`. Safe to share
  across goroutines.
- `programCache` is mutex-protected. Cache key is `<step>:<kind>.<dim_id>:<source>`
  to prevent cross-mapping collisions (the original bug that
  motivated the property tests in
  `internal/calculator/concurrency_property_test.go`).
- `MemoCache` uses a `sync.Map` for lookup-heavy workloads.
- `DiskCache` lets the SQLite driver handle locking (single-writer,
  many-reader by default).

## Build + test

```bash
# Standard test loop
go test -race ./...

# Catalog verifier — runs every kind end-to-end against pricing.c3x.dev
go run ./cmd/verify_catalog

# Strict mode: also fail on snapshot DRIFT or missing fixtures
go run ./cmd/verify_catalog -strict

# Benchmarks
go test -bench=. -benchmem -run='^$' ./...

# Fuzz tests (seeds run in normal test mode; -fuzz to actually fuzz)
go test ./internal/expr/... -fuzz=FuzzCompile -fuzztime=30s
```

## Adding things

| What | Where | Effort |
|---|---|---|
| A new resource kind | One TOML under `resources/<provider>/<kind>.toml` with a `[fixture]` block | 30 min |
| A new recommendation | `internal/recommend/rules_<cloud>.go` + test | 1 hour |
| A new output format | `internal/render/<format>.go` + dispatcher case + format-name parsing | 2 hours |
| A new forge poster | Implement `comment.Poster` in `internal/comment/<forge>.go` + add subcommand under `cmd/c3x/comment.go` | half day |
| A new parser | `internal/parser/<format>/` + dispatcher case in `internal/parser/parser.go` | 1-3 days |

See `CONTRIBUTING.md` for the workflow checklist (verifier, snapshot
regen, race tests).
