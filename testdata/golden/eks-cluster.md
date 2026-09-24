## c3x estimate

### 📦 `aws_eks_cluster.main`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Control plane (EKS classic) | 730 hours | $0 | $0 | stub |

### 📦 `aws_instance.worker[0]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |
| Root EBS volume | 100 GB-month | $0 | $0 | stub |

### 📦 `aws_instance.worker[1]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |
| Root EBS volume | 100 GB-month | $0 | $0 | stub |

### 📦 `aws_instance.worker[2]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |
| Root EBS volume | 100 GB-month | $0 | $0 | stub |

### 📦 `aws_instance.worker[3]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |
| Root EBS volume | 100 GB-month | $0 | $0 | stub |

### 📦 `aws_lb.ingress`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Load balancer | 730 hours | $0 | $0 | stub |
| Load balancer capacity units | 500 LCU-hours | $0 | $0 | stub |

### 📦 `aws_s3_bucket.artifacts`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Standard storage | 200 GB-month | $0 | $0 | stub |
| PUT, COPY, POST, LIST requests | 50000 requests | $0 | $0 | stub |
| GET, SELECT, and other requests | 500000 requests | $0 | $0 | stub |

### 📦 `aws_s3_bucket.state`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Standard storage | 10 GB-month | $0 | $0 | stub |
| PUT, COPY, POST, LIST requests | 1000 requests | $0 | $0 | stub |
| GET, SELECT, and other requests | 10000 requests | $0 | $0 | stub |

### 📦 `aws_cloudwatch_log_group.cluster`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Data ingested | 30 GB | $0 | $0 | stub |
| Data stored (archive) | 60 GB-month | $0 | $0 | stub |
| Logs Insights queries | 5 GB scanned | $0 | $0 | stub |

**Project total: $0/mo**

> ⚠️ **20 caveats:** parts of this estimate rest on assumptions (a price from another region, usage not provided, an attribute that could not be evaluated), so it may misstate the real cost.

<details><summary>Caveats</summary>

| Resource | Line | Caveat |
|---|---|---|
| `aws_eks_cluster.main` | Control plane (EKS classic) | offline stub price, not a real rate |
| `aws_instance.worker[0]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.worker[0]` | Root EBS volume | offline stub price, not a real rate |
| `aws_instance.worker[1]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.worker[1]` | Root EBS volume | offline stub price, not a real rate |
| `aws_instance.worker[2]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.worker[2]` | Root EBS volume | offline stub price, not a real rate |
| `aws_instance.worker[3]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.worker[3]` | Root EBS volume | offline stub price, not a real rate |
| `aws_lb.ingress` | Load balancer | offline stub price, not a real rate |
| `aws_lb.ingress` | Load balancer capacity units | offline stub price, not a real rate |
| `aws_s3_bucket.artifacts` | Standard storage | offline stub price, not a real rate |
| `aws_s3_bucket.artifacts` | PUT, COPY, POST, LIST requests | offline stub price, not a real rate |
| `aws_s3_bucket.artifacts` | GET, SELECT, and other requests | offline stub price, not a real rate |
| `aws_s3_bucket.state` | Standard storage | offline stub price, not a real rate |
| `aws_s3_bucket.state` | PUT, COPY, POST, LIST requests | offline stub price, not a real rate |
| `aws_s3_bucket.state` | GET, SELECT, and other requests | offline stub price, not a real rate |
| `aws_cloudwatch_log_group.cluster` | Data ingested | offline stub price, not a real rate |
| `aws_cloudwatch_log_group.cluster` | Data stored (archive) | offline stub price, not a real rate |
| `aws_cloudwatch_log_group.cluster` | Logs Insights queries | offline stub price, not a real rate |

</details>
