## c3x estimate

### ⚠️ `aws_eip.nat[0]`  —  $3.65/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Public IPv4 address-hour | 730 hours | $0.005 | $3.65 | static |

### ⚠️ `aws_eip.nat[1]`  —  $3.65/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Public IPv4 address-hour | 730 hours | $0.005 | $3.65 | static |

### ⚠️ `aws_eip.nat[2]`  —  $3.65/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Public IPv4 address-hour | 730 hours | $0.005 | $3.65 | static |

### 📦 `aws_nat_gateway.main[0]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Gateway | 730 hours | $0 | $0 | stub |
| Data processed | 200 GB | $0 | $0 | stub |

### 📦 `aws_nat_gateway.main[1]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Gateway | 730 hours | $0 | $0 | stub |
| Data processed | 200 GB | $0 | $0 | stub |

### 📦 `aws_nat_gateway.main[2]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Gateway | 730 hours | $0 | $0 | stub |
| Data processed | 200 GB | $0 | $0 | stub |

### 📦 `aws_lb.edge`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Load balancer | 730 hours | $0 | $0 | stub |
| Load balancer capacity units | 1000 LCU-hours | $0 | $0 | stub |

### 📦 `aws_db_instance.primary`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Database instance | 730 hours | $0 | $0 | stub |
| Storage | 100 GB-month | $0 | $0 | stub |

### 📦 `aws_instance.api["api"]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |

### 📦 `aws_instance.api["scheduler"]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |

### 📦 `aws_instance.api["worker"]`  —  $0/mo

| Dimension | Quantity | Unit rate | Monthly | Source |
|---|---:|---:|---:|---|
| Instance usage (Linux/UNIX, on-demand) | 730 hours | $0 | $0 | stub |

**Project total: $10.95/mo**

> ⚠️ **13 caveats:** parts of this estimate rest on assumptions (a price from another region, usage not provided, an attribute that could not be evaluated), so it may misstate the real cost.

<details><summary>Caveats</summary>

| Resource | Line | Caveat |
|---|---|---|
| `aws_nat_gateway.main[0]` | Gateway | offline stub price, not a real rate |
| `aws_nat_gateway.main[0]` | Data processed | offline stub price, not a real rate |
| `aws_nat_gateway.main[1]` | Gateway | offline stub price, not a real rate |
| `aws_nat_gateway.main[1]` | Data processed | offline stub price, not a real rate |
| `aws_nat_gateway.main[2]` | Gateway | offline stub price, not a real rate |
| `aws_nat_gateway.main[2]` | Data processed | offline stub price, not a real rate |
| `aws_lb.edge` | Load balancer | offline stub price, not a real rate |
| `aws_lb.edge` | Load balancer capacity units | offline stub price, not a real rate |
| `aws_db_instance.primary` | Database instance | offline stub price, not a real rate |
| `aws_db_instance.primary` | Storage | offline stub price, not a real rate |
| `aws_instance.api["api"]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.api["scheduler"]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |
| `aws_instance.api["worker"]` | Instance usage (Linux/UNIX, on-demand) | offline stub price, not a real rate |

</details>
