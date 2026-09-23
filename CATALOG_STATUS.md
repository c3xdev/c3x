# Catalog verification status

The harness builds a representative resource per kind, runs the engine
against the live API, and reports per-resource pricing health:

```bash
go run ./cmd/verify_catalog          # health of every kind, against live prices
go run ./cmd/gen_catalog_doc > docs/catalog.md   # regenerate the per-kind matrix
```

## Current state

Verified against `pricing.c3x.dev` on 2026-09-23:

| Result | Kinds |
|---|---:|
| **LIVE** (priced from the upstream API, tracks vendor price changes) | 152 |
| **STATIC** (priced from an inline rate, does not track upstream changes) | 111 |
| **FREE** (not charged at the resource level) | 1,077 |
| **ZERO / DRIFT / NOFIX / STALE / ERRORED** | 0 |
| **Total** | **1,340** |

So 263 kinds carry a price and the remaining 1,077 are legitimately free.

The per-kind matrix is generated at [`docs/catalog.md`](docs/catalog.md);
it is the list, and this page is only the summary. Counts are deliberately
stated once here rather than repeated per section, because the previous
version of this file carried the same numbers in six places and they
drifted apart from each other and from reality.

### Two classifications, counted differently

`docs/catalog.md` reports **176 LIVE / 87 STATIC** for the same 1,340
kinds. That is not a contradiction, the two tools measure different
things:

- `gen_catalog_doc` classifies from the **TOML declaration**, via
  `Registry.HasStaticRate`: does this kind declare an inline rate at all.
- `verify_catalog` classifies from an **actual run**: what the fixture's
  line items really priced through.

The verifier's numbers are the operational truth, which is why they are
the ones quoted above. The gap between the two (24 kinds) is worth
reconciling on its own; it means some kinds declare no inline rate yet
still price through one at runtime.

## Output legend

| Marker | Meaning |
|---|---|
| `[OK]` | Non-zero estimate against the live API |
| `[STATIC]` | Non-zero estimate via an inline TOML literal |
| `[FREE]` | Known-free resource (parent-rolled or always free) |
| `[ZERO]` | No priced line items. A regression sentinel, fails CI |
| `[DRIFT]` | Priced, but far from the recorded snapshot |
| `[STALE]` | A STATIC rate whose `last_verified` date is over six months old |
| `[ERR]` | The engine returned an error |

Every STATIC fixture carries a `last_verified` date the verifier
enforces, so an inline rate cannot sit unchecked indefinitely. The
per-kind list of upstream gaps, with the GraphQL probes that returned
empty, is in [`docs/upstream-gaps.md`](docs/upstream-gaps.md).

## Converting a STATIC entry to live

```bash
# 1. Probe what the upstream exposes for the service.
curl -sS -X POST https://pricing.c3x.dev/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"{products(filter:{vendorName:\"<aws|azure|gcp>\",service:\"<service>\"},limit:5){productFamily attributes{key value} prices{USD purchaseOption unit}}}"}' | jq

# 2. Find a tuple that identifies exactly one product. Check it in more
#    than one region: usagetype carries a region prefix outside
#    us-east-1 (EU-InstanceUsage:...), so pinning it matches nothing
#    elsewhere. Prefer an attribute that is stable across regions.

# 3. Point the resource TOML at it:
#    service         = "..."
#    product_family  = "..."
#    region          = "global"   (if the upstream entry has no regionCode)
#    purchase_option = "any"      (if the entries carry no purchaseOption)
#    attribute_filters = [{ key = "...", const = "..." }]

# 4. Re-run the verifier for that kind.
go run ./cmd/verify_catalog | grep <kind>
```

A filter that matches more than one product is the failure mode to watch
for: the engine takes the highest non-zero price among the matches, so an
under-specified filter silently prices the dearer variant rather than
erroring. `go run ./cmd/verify_catalog -audit` reports mappings whose
matches span a suspicious price range.

## Engine behaviour the catalog relies on

- **Max non-zero pricing.** Tiered services (CloudFront, S3 requests)
  return several rates per product. The HTTP source takes the largest
  non-zero, so c3x quotes the conservative first-tier on-demand rate
  rather than the cheapest committed tier.
- **`purchase_option = "any"`.** When upstream price entries carry no
  purchaseOption label, a mapping opts out of that filter so the query
  still returns rows.
- **Global-region override.** For products billed without a regionCode
  (CloudFront, Pub/Sub message delivery, GKE control plane), a mapping
  sets `region = "global"` to omit the region clause.
- **Attribute inheritance.** Some costs are decided by an attribute
  declared on a different resource, such as Aurora's `storage_type`,
  which sits on the cluster while the hours are billed on the instance.
  The parser copies those across before pricing
  (`internal/parser/inherit.go`), which keeps the catalog on plain
  attribute reads that every released client understands.
- **Three-layer cache.** Memo (in-process), then SQLite (7-day TTL),
  then HTTP. Most runs against a populated cache finish in under 50ms.
