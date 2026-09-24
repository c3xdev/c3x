# Upstream pricing gaps

This document tracks every resource in `resources/` whose price is
inlined as a literal TOML `rate = "<value>"` rather than queried from
`pricing.c3x.dev`. Each entry exists because the upstream pricing
catalog does not currently expose the right meter — the gap is in the
backend's coverage of vendor catalogs, not in `c3x-go`.

The intended fix path for every entry is to enrich `c3x-pricing-api`
(the backend) so the meter becomes queryable, then convert the TOML
back to a `price("…")` call. Editing the TOML alone cannot move a
STATIC entry to LIVE.

Last verified: 2026-06-08. Run `go run ./cmd/verify_catalog -live` to
regenerate the counts (currently 52 LIVE / 25 STATIC / 2 FREE).

## How to validate a gap

Each entry below names the GraphQL `Service` + `ProductFamily` (and
attribute filters) that *should* produce a price. Probe the backend
with:

```bash
curl -s 'https://pricing.c3x.dev/graphql' \
  -H 'content-type: application/json' \
  -d '{"query":"{ products(filter: { vendorName: \"aws\", service: \"<Service>\", productFamily: \"<Family>\" }) { prices { USD } attributes { key value } } }"}'
```

An empty `products` array is the signal the gap is real. Once the
backend exposes the meter, the rate moves from inline literal to
`price("<dim>")`.

## AWS (4)

| Kind | Service / ProductFamily | Why STATIC | Backend fix |
|---|---|---|---|
| `aws_eip` | `AmazonEC2 / IP Address` | Public IPv4-hour meter not exposed; the upstream catalog only carries EC2 instance pricing, not the IPv4-hour line item AWS introduced Feb-2024 | Add the `IP Address` family with `groupDescription = "Hourly charge for In-use public IPv4 addresses"` attribute |
| `aws_eks_cluster` | `AmazonEKS / Compute` | Only AutoMode SKUs are indexed; the classic $0.10/hr control-plane fee has no addressable product | Add `usagetype` pattern `*-AmazonEKS-Hours` |
| `aws_athena_workgroup` | `AmazonAthena / Athena` | Per-TB-scanned pricing has no `ProductFamily` set in the upstream feed; rows look like `{operation: "QueryGB"}` only | Backfill `productFamily = "Query"` on Athena SKUs |
| `aws_sagemaker_endpoint` | `AmazonSageMaker / ML Instance` | The ML-instance SKUs are uploaded but `instanceType` attribute is empty so filters can't match | Backfill `instanceType` from `usagetype` regex |
| `aws_dax_cluster` | `AmazonDAX` | Service is entirely absent from the upstream scrape (probed 2026-06-08, 0 products); inline rates cover t3 + r4/r5 node families | Add a scraper for the DynamoDB-DAX SKUs (`product[].attributes.servicecode = "AmazonDAX"` in AWS Bulk Pricing) |
| `aws_memorydb_cluster` | `AmazonMemoryDB` | Service is entirely absent from the upstream scrape (probed 2026-06-08, 0 products); inline rates cover t4g + r6g/r7g node families | Add a scraper for the MemoryDB SKUs |
| `aws_emr_cluster` | `ElasticMapReduce` | Service is entirely absent from the upstream scrape (probed 2026-06-08, 0 products); inline service-charge rate of $0.048/node-hour models the EMR markup on top of underlying EC2 | Add a scraper for the EMR SKUs |
| `aws_db_proxy` | `AmazonRDS / Database Proxy` | No `DB Proxy`/`Database Proxy` productFamily found in the upstream catalog (probed 2026-06-08); inline rate of $0.015/vCPU-hour from the published RDS Proxy pricing page | Add the `Database Proxy` family or backfill `productFamily` on the existing RDS Proxy SKUs |

## Azure (8)

| Kind | Service / Meter | Why STATIC | Backend fix |
|---|---|---|---|
| `azurerm_container_registry` | `Container Registry / Standard Registry Unit` | Standard/Basic-tier connected-registry meters are not in the Retail Prices snapshot | Pull from Retail Prices API with `serviceName = "Container Registry"` |
| `azurerm_data_factory` | `Azure Data Factory v2 / Pipeline Activity` | Activity-run and DIU meters are not currently scraped | Add scraper for `serviceName = "Azure Data Factory v2"` |
| `azurerm_eventhub_namespace` | `Event Hubs / Throughput Unit` | Standard Throughput Unit hour meter not present | Add `meterName = "Throughput Unit"` rows |
| `azurerm_firewall` | `Azure Firewall / Standard Data Processed` | Standard data-processed GB meter not exposed (only deployment-hour is) | Backfill `meterName = "Standard Data Processed"` |
| `azurerm_key_vault` | `Key Vault / Operations` | Standard operations per-10k meter not exposed | Add `meterName = "Operations"` rows |
| `azurerm_logic_app_workflow` | `Logic Apps / Consumption Actions` | Consumption-tier Actions meter not indexed (Standard tier IS indexed but billed via App Service Plan) | Add `meterName = "Standard Actions"` (consumption rows) |
| `azurerm_servicebus_namespace` | `Service Bus / Standard Base` | Standard-namespace base charge has no row | Add `meterName = "Standard Base Unit"` |
| `azurerm_synapse_workspace` | `Synapse Analytics / Serverless SQL` | Serverless on-demand per-TB pricing absent | Add `meterName = "Serverless"` rows |

## GCP (12)

| Kind | Service / SKU | Why STATIC | Backend fix |
|---|---|---|---|
| `google_artifact_registry_repository` | `Artifact Registry / Storage` | Storage GB-month meter not present in Cloud Billing Catalog scrape | Add `service = "Artifact Registry"` SKU group |
| `google_bigquery_dataset` | `BigQuery / Analysis` | On-demand per-TB scan rate missing (only storage is present) | Backfill `category = "Analysis"` SKUs |
| `google_cloud_tasks_queue` | `Cloud Tasks / Operations` | Operations meter not exposed | Add `service = "Cloud Tasks"` SKUs |
| `google_cloudfunctions2_function` | `Cloud Functions / Invocations` | 2nd-gen functions bill through Cloud Run but the invocation meter isn't routed | Map invocations through Cloud Run scrape |
| `google_compute_router_nat` | `Compute Engine / NAT VM-hours` + `NAT GB` | Per-VM-hour and per-GB meters are present in some regions, missing in others — not safe to query without region-by-region coverage | Reconcile region coverage |
| `google_dataflow_job` | `Dataflow / Worker vCPU` + `Worker RAM` | Streaming/batch worker hour meters not exposed | Add `service = "Cloud Dataflow"` worker SKUs |
| `google_dataproc_cluster` | `Dataproc / Service Charge` | Per-vCPU-hour service charge (on top of Compute Engine) not exposed | Add `category = "Service Charge"` SKUs |
| `google_filestore_instance` | `Filestore / Standard` | Standard-tier capacity meter not present (Premium IS) | Backfill Standard SKU |
| `google_logging_project_bucket` | `Cloud Logging / Ingestion` | GB ingestion meter not exposed | Add `service = "Cloud Logging"` |
| `google_redis_instance` | `Memorystore / Standard HA` | Capacity meters absent | Add `service = "Memorystore for Redis"` SKUs |
| `google_secret_manager_secret` | `Secret Manager / Active Versions` | Per-version meter not exposed | Add `service = "Secret Manager"` |
| `google_spanner_instance` | `Spanner / Processing Unit` | Processing-unit pricing not exposed (only Node pricing partially is) | Backfill PU meter |

## Conversion checklist

When the backend ships a fix, the workflow to move an entry from
STATIC → LIVE is:

1. Verify the new meter responds to the GraphQL probe above with a
   non-empty `products` array.
2. In the resource's TOML, replace the literal `rate = "0.045"`
   (or whatever) with `rate = 'price("<dimension>")'`.
3. Add or update `[mappings.<dimension>]` with the `service`,
   `product_family`, and `attribute_filters` derived from the
   GraphQL response.
4. Run `go run ./cmd/verify_catalog -live`. The kind should report
   `[OK]`.
5. Update the `[STATIC] / [LIVE]` counts in `CATALOG_STATUS.md`.
6. Remove the row from this document.

## Out of scope

The two `[FREE]` resources (`aws_acm_certificate`,
`aws_lb_target_group`) are intentionally rated $0 — they're billed
through parent resources or genuinely free. They are not gaps and
should not move to LIVE.
