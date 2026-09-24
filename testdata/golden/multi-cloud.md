## c3x estimate

### 📦 `aws_cloudfront_distribution.edge`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Data transfer (US/Canada) | 5000 GB | $0 | $0 | stub |
| Data transfer (Europe) | 0 GB | $0 | $0 | stub |
| Data transfer (Japan) | 0 GB | $0 | $0 | stub |

### 📦 `aws_s3_bucket.origin`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Standard storage | 500 GB-month | $0 | $0 | stub |
| PUT, COPY, POST, LIST requests | 100000 requests | $0 | $0 | stub |
| GET, SELECT, and other requests | 1000000 requests | $0 | $0 | stub |

### 📦 `azurerm_postgresql_flexible_server.main`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Database compute (D2ads_v5, 2 vCore) | 1460 vCore-hours | $0 | $0 | stub |

### 📦 `google_storage_bucket.backups`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Standard Storage (US Regional) | 2000 GB-month | $0 | $0 | stub |

**Project total: $0/mo**

> ⚠️ **8 caveats:** parts of this estimate rest on assumptions (a price from another region, usage not provided, an attribute that could not be evaluated), so it may misstate the real cost.

<details><summary>Caveats</summary>

| Resource | Line | Caveat |
|---|---|---|
| `aws_cloudfront_distribution.edge` | Data transfer (US/Canada) | offline stub price, not a real rate |
| `aws_cloudfront_distribution.edge` | Data transfer (Europe) | offline stub price, not a real rate |
| `aws_cloudfront_distribution.edge` | Data transfer (Japan) | offline stub price, not a real rate |
| `aws_s3_bucket.origin` | Standard storage | offline stub price, not a real rate |
| `aws_s3_bucket.origin` | PUT, COPY, POST, LIST requests | offline stub price, not a real rate |
| `aws_s3_bucket.origin` | GET, SELECT, and other requests | offline stub price, not a real rate |
| `azurerm_postgresql_flexible_server.main` | Database compute (D2ads_v5, 2 vCore) | offline stub price, not a real rate |
| `google_storage_bucket.backups` | Standard Storage (US Regional) | offline stub price, not a real rate |

</details>
