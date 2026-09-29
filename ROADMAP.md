# c3x — Roadmap

Planned features and coverage work for c3x.

Effort scale: **S** ≤ 1 day · **M** 1–3 days · **L** ≈ 1 week ·
**XL** > 1 week.

Tag legend at end of each line:
- 🅻 lands changes in `c3x-go` (this repo) — actionable here
- 🅱 lands changes in the `c3x-pricing-api` backend — blocked on
  separate repo
- 🅲 cross-cutting (touches both)

---

## A. Resource coverage (132 priced → target ≥ 250)

### A.1 Backend enrichment (`c3x-pricing-api`) 🅱
Single largest gap. Documented in `docs/upstream-gaps.md`.

- [ ] **A.1.1** [**XL**] Scrape AWS Bulk Pricing JSON for 11
  missing-service offers (AmazonDAX, AmazonMemoryDB, ElasticMapReduce,
  AWSAppRunner, AWSAppSync, AWSGlobalAccelerator, AWSAmplify,
  AmazonCognito, AWSXRay, AmazonStorageGateway, AmazonRoute53 health
  checks).
- [ ] **A.1.2** [**M**] Backfill missing attributes — SageMaker
  `instanceType`, EKS classic control-plane `usagetype`, Athena
  `productFamily`.
- [ ] **A.1.3** [**L**] Hand-curated rate table for meters never in
  bulk pricing (EIP IPv4-hour, CW Events, Cognito M2M).
- [ ] **A.1.4** [**L**] Azure Retail Prices additions: Cosmos DB
  Standard provisioned RU, Key Vault Operations, Logic Apps
  Consumption Actions, Container Registry Standard.
- [ ] **A.1.5** [**M**] GCP Cloud Billing Catalog additions:
  Memorystore capacity, BigQuery on-demand analysis, Cloud Run
  requests.

### A.2 New TOMLs against existing backend coverage 🅻

- [ ] **A.2.1** [**M**] AWS: `aws_workspaces_workspace`,
  `aws_glue_crawler`, `aws_glue_database`, `aws_msk_serverless_cluster`,
  `aws_apigatewayv2_route` (WebSocket data), `aws_directconnect_lag`,
  `aws_directconnect_gateway`, `aws_lambda_function_url` (FREE),
  `aws_chatbot_slack_channel_configuration` (FREE).
- [ ] **A.2.2** [**M**] Azure: `azurerm_container_app`,
  `azurerm_data_lake_storage_gen2_filesystem`, `azurerm_search_service`,
  `azurerm_synapse_sql_pool`, `azurerm_databricks_workspace`.
- [ ] **A.2.3** [**M**] GCP: `google_compute_instance_from_template`,
  `google_dataproc_metastore_service`, `google_bigtable_instance`,
  `google_alloydb_cluster`, `google_certificate_manager_certificate`.
- [ ] **A.2.4** [**L**] Audit each provider's resource list and
  fill remaining gaps where the backend has coverage.

### A.3 Catalog hygiene 🅻

- [ ] **A.3.1** [**S**] Add `last_verified` to `[fixture]` block;
  verifier warns on >6mo stale entries.
- [ ] **A.3.2** [**S**] Scheduled `verify_catalog -strict` job
  catching upstream rate drift.

---

## B. CI / forge integrations (GitHub only → +4) 🅻

The `comment.Poster` interface is the seam.

- [ ] **B.1** [**M**] GitLab MR comment poster
  (`internal/comment/gitlab.go`) — marker-based update.
- [ ] **B.2** [**M**] Bitbucket Cloud PR comment poster.
- [ ] **B.3** [**M**] Azure DevOps PR comment poster.
- [ ] **B.4** [**S**] Auto-detect logic per forge (CI env vars).
- [ ] **B.5** [**L**] Atlantis integration plugin or wrapper image.
- [ ] **B.6** [**S**] Stub-server tests for each new poster.

---

## C. Multi-currency 🅻

- [ ] **C.1** [**M**] FX conversion layer in `internal/pricing` (ECB
  or Frankfurter API, 24h cache).
- [ ] **C.2** [**S**] Broaden `domain.Currency` (EUR, GBP, JPY,
  CAD, AUD).
- [ ] **C.3** [**S**] Locale-aware formatting in `internal/render`.
- [ ] **C.4** [**S**] CLI flag `--currency EUR` wired through
  resolved config.
- [ ] **C.5** [**M**] Per-currency snapshot tests pinning rounding.

---

## D. Policy / budget engine (OPA / Rego) 🅻

- [ ] **D.1** [**L**] Integrate `open-policy-agent/opa` as a library.
  `c3x policy eval --policy <rego> --estimate <est.json>`.
- [ ] **D.2** [**M**] Policy data model: `input.resources[*]`,
  `input.total`, `input.diff_vs_baseline`.
- [ ] **D.3** [**S**] Wire policy eval into `c3x diff`.
- [ ] **D.4** [**M**] Sample policies under `examples/policies/`.
- [ ] **D.5** [**M**] Policy corpus tests.

---

## E. Module support (registry + remote) 🅻

- [ ] **E.1** [**M**] Terraform Registry fetcher
  (`source = "terraform-aws-modules/vpc/aws"`).
- [ ] **E.2** [**M**] Git module fetcher (`git::https://...`,
  `git::ssh://...`).
- [ ] **E.3** [**S**] Module cache at `~/.cache/c3x/modules/`.
- [ ] **E.4** [**M**] Version-constraint resolution
  (`version = "~> 3.0"`).
- [ ] **E.5** [**S**] Tests pinning `terraform-aws-modules/vpc/aws`
  at a known version.

---

## F. Output formats 🅻

- [ ] **F.1** [**M**] HTML output (`render.FormatHTML`).
- [ ] **F.2** [**S**] JUnit XML output (`render.FormatJUnit`).
- [ ] **F.3** [**S**] SARIF output for security surfaces.
- [ ] **F.4** [**S**] CSV output.

---

## G. Release tooling & distribution 🅻 (user-deferred to last)

- [ ] **G.1** [**M**] `.goreleaser.yaml` — multi-arch, signed.
- [ ] **G.2** [**S**] GitHub Actions release workflow.
- [ ] **G.3** [**S**] Homebrew tap.
- [ ] **G.4** [**S**] Docker images (ghcr.io/c3xdev/c3x).
- [ ] **G.5** [**S**] GitHub Action wrapper (`c3xdev/c3x-action`).
- [ ] **G.6** [**S**] `asdf` plugin.
- [ ] **G.7** [**M**] Stable PR comment template so adopting c3x in
  CI is one `uses:` change.

---

## H. Documentation 🅻

- [ ] **H.1** [**M**] `ARCHITECTURE.md` — package graph, catalog
  loading, rule composition.
- [ ] **H.2** [**S**] `CONTRIBUTING.md` — add a TOML, add a rule,
  run the verifier.
- [ ] **H.3** [**L**] Docs site (Docusaurus/VitePress) at `c3x.dev`.
- [ ] **H.4** [**S**] Auto-generated resource catalog page from
  TOMLs (status, dimensions).

---

## I. Smaller gaps 🅻

- [ ] **I.1** [**S**] "Δ" column in PR comment.
- [ ] **I.2** [**M**] `c3x configure` — interactive first-run setup.
- [ ] **I.3** [**S**] `c3x doctor` — pre-flight checks.
- [ ] **I.4** [**S**] `c3x supported-resources` — list catalog with
  per-kind status.
- [ ] **I.5** [**M**] Prometheus `/metrics` endpoint for self-hosted.
- [ ] **I.6** [**S**] `--show-skipped` flag on `estimate`.

---

## K. Usage sync 🅻

Fill `c3x-usage.yml` from cloud metrics so usage-driven lines are not $0.
Design: [`docs/design/usage-sync.md`](docs/design/usage-sync.md), issue #97.

- [ ] **K.1** [**M**] Skeleton + S3 storage (CloudWatch), address mapping
  from Terraform state, synced + hand-written files with precedence.
- [ ] **K.2** [**M**] S3 requests (request metrics where enabled, else
  Cost Explorer).
- [ ] **K.3** [**S**] Lambda and NAT gateway (CloudWatch).
- [ ] **K.4** [**M**] CloudFront transfer by region group (Cost Explorer).
- [ ] **K.5** [**L**] GCP: Cloud Run v2 and GCS.

---

## J. Stretch (post-parity)

- [ ] **J.1** [**XL**] Bicep / ARM parser.
- [ ] **J.2** [**XL**] Pulumi state file parser.
- [ ] **J.3** [**XL**] CDK CloudFormation support.
- [ ] **J.4** [**L**] DigitalOcean / Cloudflare / Vercel providers.

---

## Execution order

1. A.1.1 + A.1.2 (backend) — biggest visible gap [🅱 blocked here]
2. G — without release, no external adoption [user-deferred]
3. B.1 + B.2 — broaden CI surface
4. C — small, high-perception
5. F.1 + F.2 — fill CI integration formats
6. D — high-perception parity feature
7. E — production users will need it
8. H — gates external adoption alongside G
9. A.2 — fill the long tail
10. I, J — polish

**Rough total:** ~3-4 person-months. Backend work is the longest
single thread; most c3x-go items are ≤1 week each.
