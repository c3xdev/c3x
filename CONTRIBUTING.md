# Contributing to c3x

Read `ARCHITECTURE.md` first for the package map. This file is the
short workflow checklist for landing the three most common kinds of
change.

## Quick reference

```bash
git clone https://github.com/c3xdev/c3x.git
cd c3x
go test -race ./...            # full suite
go run ./cmd/verify_catalog    # catalog health against pricing.c3x.dev
```

The repo enforces `gofmt`, `go vet`, and `depguard` (module
boundaries). CI runs all three; locally:

```bash
gofmt -s -w .
go vet ./...
```

## Adding a new resource kind (most common contribution)

1. Create `resources/<provider>/<kind>.toml`. Skeleton:

   ```toml
   kind         = "aws_widget"
   display_name = "AWS Widget"
   provider     = "aws"

   [mappings.widget]
   service        = "AmazonWidget"
   product_family = "Widget"
   attribute_filters = [
     { key = "instanceType", expr = 'default(size, "small")' },
   ]

   [[dimensions]]
   id       = "widget_hours"
   label    = "Widget"
   unit     = "hours"
   quantity = "monthly_hours()"
   rate     = 'price("widget")'

   [fixture]
   attributes = { size = "small" }
   expected_monthly_cost = 0
   tolerance = 0.05
   ```

2. Probe the upstream backend to find the right mapping shape:

   ```bash
   curl -s 'https://pricing.c3x.dev/graphql' \
     -H 'content-type: application/json' \
     -d '{"query":"{ products(filter: { vendorName: \"aws\", service: \"AmazonWidget\", region: \"us-east-1\" }) { productFamily attributes { key value } prices { USD } } }"}'
   ```

3. Run the verifier to compute the actual monthly cost:

   ```bash
   go run ./cmd/verify_catalog | grep aws_widget
   ```

4. Check that value against the vendor's public pricing page or
   calculator for the same configuration and region, then set the
   fixture's `expected_monthly_cost` to the vendor's number, not to
   c3x's output. A fixture copied from the engine only proves the
   engine agrees with itself. The verifier then enforces it on every
   run as a snapshot.

5. Add the kind to the appropriate parser's attribute extractor if
   the Terraform attribute names need normalization (see
   `internal/parser/terraform/resources.go` for examples).

6. Land the TOML in
   [c3x-pricing-api](https://github.com/c3xdev/c3x-pricing-api)'s
   `catalog/` first. That repository is the source of truth, and
   pricing.c3x.dev serves its catalog to every CLI. Then copy it here
   with `scripts/sync-catalog.sh`. The `catalog-sync` check fails
   while the two copies differ.

If the upstream returns no products, fall back to inline rates:

```toml
rate = "0.05"   # static — document the gap in docs/upstream-gaps.md
```

`exact = true` in the fixture instead of `tolerance` so any drift
fails CI loudly.

## Adding a recommendation rule

1. Pick `rules_aws.go`, `rules_azure.go`, or `rules_gcp.go` based
   on the cloud. (Cross-provider rules go in `recommend.go`.)
2. Implement `Rule`:

   ```go
   type WidgetUnderutilization struct{}

   func (WidgetUnderutilization) Name() string { return "aws.widget.under" }

   func (WidgetUnderutilization) Propose(r domain.Resource) []Proposal {
       if r.Ref.Kind != "aws_widget" {
           return nil
       }
       // … return one or more Proposals
   }
   ```

3. Register it in the per-cloud factory (`AWSRules()`, etc.).
4. Add tests in `recommend_test.go` and the table-driven
   `rules_test.go`.

For cross-resource recommendations use `TreeRule.ProposeTree` and
return a `TreeProposal` with the `Changes` map. The engine
re-estimates the modified set; positive net delta becomes a
recommendation.

## Adding an output format

1. `internal/render/<format>.go` with `Render<Format>(est) string`
   (or `(string, error)` for serialization formats).
2. Add the constant to `internal/render/render.go`'s `Format` enum
   + `String()` + `ParseFormat()`.
3. Add a `case` to `Render()` and `RenderDiff()` in
   `internal/render/dispatch.go`.
4. Test the rendering against a hand-crafted `domain.Estimate` —
   no full pipeline needed.

## Adding a forge poster (PR comments on GitLab / Bitbucket / …)

1. Implement `comment.Poster` in `internal/comment/<forge>.go`.
   Reuse the marker pattern (`comment.Marker`).
2. Add an env-driven `AutoDetect<Forge>()` helper.
3. Add a subcommand in `cmd/c3x/comment.go` that wires flags +
   constructor + dispatch.
4. Test with `httptest.NewServer` stubbing the forge's API.

## Writing tests

- `go test -race ./...` must pass before submitting.
- Property tests live next to the unit tests
  (`recommend_test.go`, `concurrency_property_test.go`).
- Fuzz tests go in `<package>/fuzz_test.go` with seed corpora.
- Golden tests update with `-update`:

  ```bash
  go test ./cmd/c3x/... -run TestEstimateGolden -update
  ```

- Catalog snapshots regenerate by editing the affected TOML's
  `[fixture]` block + running the verifier.

## Catalog drift policy

The verifier reports `[DRIFT]` when a kind's computed cost moves
outside its snapshot tolerance. Drift is not always a bug — vendor
prices change. When you see drift:

1. Confirm against the vendor's public pricing page that the new
   rate is correct.
2. Update the affected TOML's `expected_monthly_cost`.
3. Bump `last_verified` (when we add that field — A.3.1 on the
   roadmap).
4. Commit with a message like `catalog: refresh aws_eks rate`.

## Changing the CLI surface (flags, env vars, config keys)

The public CLI surface is a contract. See `COMPATIBILITY.md` for the
full policy; the short version:

- Adding a flag: fine. Regenerate the surface golden with
  `UPDATE_CLI_SURFACE=1 go test ./cmd/c3x/ -run TestCLISurface` and
  commit the one-line diff.
- Removing or renaming a flag: never silently. Deprecate it first
  (`cmd.Flags().MarkDeprecated(...)`), note it in the CHANGELOG, and
  remove it no sooner than the next minor release.

`TestCLISurface` fails CI on any un-regenerated surface change, so a
removal shows up as a golden deletion in review.

## Commit conventions

Single-purpose commits. Conventional Commits style preferred:

```
catalog: add aws_workspaces_workspace TOML
parser: handle CFN !Sub with mid-string variables
calculator: fix cache key collision on multi-mapping filters
```

PR descriptions should link to the relevant `ROADMAP.md` item if
one exists.
