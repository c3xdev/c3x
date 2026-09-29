# Design: `c3x usage sync`

Status: accepted, 2026-09-29. Implementation in progress, tracked in
[#97](https://github.com/c3xdev/c3x/issues/97).

## Problem

Usage-driven lines, such as S3 storage and requests, CloudFront transfer,
Lambda invocations and NAT data processed, are $0 with a
`usage_not_provided` caveat unless someone writes `c3x-usage.yml` by hand.
In practice nobody does, so these costs are missing from most estimates.

`c3x usage sync` fills the usage file from the cloud account's own
metrics. It is separate from `c3x pricing sync`, which warms the local
**price** cache. `usage sync` fetches **quantities**, not rates.

## Decisions

### 1. It stays out of the estimate path

- `estimate`, `diff`, `comment` and the GitHub Action never call a cloud
  API and never load cloud credentials. They only read usage files.
  "No cloud account needed" stays true for everything except this one
  command, which a user runs explicitly with their own credentials.
- Read-only API calls only.
- It refuses to run in untrusted-input mode (`--no-remote-modules` /
  `C3X_NO_REMOTE_MODULES`): parsing a pull request from a fork must never
  lead to calls made with the runner's cloud credentials.

### 2. Files: a generated snapshot and a hand-written file

```
c3x-usage.synced.yml   written by `usage sync`, never edited by hand
c3x-usage.yml          written by people, never touched by `usage sync`
```

- `usage sync` writes only the synced file and replaces it whole on
  every run. It never reads or writes the hand-written file, so a
  re-sync cannot clobber a hand edit.
- Commands that price load both. The **hand-written file wins** per
  resource address and per key; the synced file fills in the rest.
- Configuration: `usage_path` (existing) is the hand-written file.
  `synced_usage_path` (new) is the snapshot, defaulting to
  `c3x-usage.synced.yml` next to `.c3x.toml` when that file exists.
  Both follow the existing rules for project config in untrusted mode:
  paths are allowed only inside the project.

### 3. File format stays `version: 0.1` and readable by older CLIs

The synced file uses the existing schema, `resource_usage` keyed by
address. Extra information goes under new top-level keys, which the
current loader ignores (YAML decoding into the struct drops unknown
fields), so older CLIs read the quantities and skip the rest:

```yaml
version: "0.1"
synced:                        # ignored by older CLIs
  provider: aws
  window_days: 30
  generated_at: 2026-09-29T12:00:00Z
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 512.4
series:                        # per-month values, ignored by older CLIs
  aws_s3_bucket.data:
    standard_storage_gb: {2026-07: 470.1, 2026-08: 498.9, 2026-09: 512.4}
errors:                        # per resource, why a value is missing
  aws_s3_bucket.logs: "no BucketSizeBytes datapoints in the window"
```

- Keys are exactly the identifiers the catalog's quantity expressions
  read (`standard_storage_gb`, `monthly_tier_1_requests`, …). A test
  checks every key the syncer can emit against the catalog, so a
  renamed catalog key fails CI instead of silently syncing nothing.
- `series` keeps per-month values so a later `usage predict` can fit a
  trend. This proposal does not design `predict`.

### 4. Mapping a Terraform address to a cloud resource

To read a bucket's metrics, sync needs the real bucket name, not
`aws_s3_bucket.data`.

1. **Terraform state is the source of truth**: the JSON from
   `terraform show -json` (no plan argument), passed with `--state`. It
   holds every resource's real identifiers.
2. A literal in HCL (`bucket = "acme-data"`) is a fallback when no state
   is given.
3. Anything that cannot be resolved is skipped and listed under
   `errors`. Sync never guesses.

Region comes from each resource's resolved region, which the parser
already computes per provider alias and per-resource attribute.
`--region` is only a fallback for resources with none.

### 5. Partial results

A failure on one resource, for example a missing permission, no
datapoints or throttling, is recorded under `errors` for that resource
and does not stop the run. The command exits non-zero only when it
could not produce any file, or with `--strict` when any resource
failed.

### 6. Code boundaries

```
cmd/c3x/usage.go                 `usage` command group, `usage sync`
internal/usagesync/              provider-neutral: address mapping,
                                 orchestration, snapshot writer,
                                 interfaces for metric sources
internal/usagesync/aws/          the only package importing the AWS SDK
```

depguard rules:
- only `cmd/c3x` imports `internal/usagesync/...`;
- `internal/usagesync/...` does not import `pricing` or `calculator`;
- `github.com/aws/aws-sdk-go-v2/...` is allowed only in
  `internal/usagesync/aws`.

The SDK is pulled in module by module: `config` and `cloudwatch` first,
then `costexplorer` when needed. It is never the monolithic SDK.
Credentials come only from `config.LoadDefaultConfig` (environment,
shared config, SSO, instance or task role), with no custom credential
handling. `c3x doctor` reports AWS credentials and region only when a
synced usage file or `usage sync` configuration is present.

### 7. Testing

CI has no cloud credentials. The metric sources are interfaces, faked in
tests. Required tests:

- an estimate before and after sync differs on the usage-driven lines
  (a fixture with the fake source);
- the hand-written file wins over the synced file, per key;
- a re-sync leaves the hand-written file byte-identical;
- every emitted key exists in the catalog;
- untrusted mode refuses to run.

## Staged delivery

| PR | Scope |
|---|---|
| 1 | Skeleton (`usage sync`, snapshot writer, loading both files with hand-written precedence), address mapping from state, **S3 storage** (`standard_storage_gb` from CloudWatch `BucketSizeBytes`, the daily StandardStorage metric that is always available). |
| 2 | S3 requests. CloudWatch request metrics exist only for buckets with a request-metrics configuration, which is opt-in and billed. Use them where present; otherwise Cost Explorer usage types (account-level unless cost allocation by resource is enabled), recorded as such in `synced`. |
| 3 | Lambda (invocations, duration) and NAT gateway bytes: plain CloudWatch metrics. |
| 4 | CloudFront transfer by region group (needs Cost Explorer usage types). Request counts wait until the catalog prices them. |
| 5 | GCP: Cloud Run v2 (`request_count`, billable instance time) and GCS. |

## Documentation

- The README distinguishes `pricing sync` (rates) from `usage sync`
  (quantities).
- The usage docs show the two-file layout and precedence.
- ROADMAP gets a section for usage sync.
