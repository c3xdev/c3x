package calculator_test

import (
	"context"
	"sync"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

// recorder prices every lookup at $1 and keeps the queries it was asked,
// so a test can check which upstream product a configuration selects and
// what quantity each line carries, without depending on live rates.
type recorder struct {
	mu      sync.Mutex
	queries []pricing.Query
}

func (r *recorder) Lookup(_ context.Context, q pricing.Query) (decimal.Decimal, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries = append(r.queries, q)
	return decimal.NewFromInt(1), domain.PriceSourceLive, nil
}

// filter returns the value the first recorded query with the given service
// set for key.
func (r *recorder) filter(t *testing.T, service, key string) string {
	t.Helper()
	for _, q := range r.queries {
		if q.Service != service {
			continue
		}
		for _, kv := range q.AttributeFilters {
			if kv.Key == key {
				return kv.Value
			}
		}
	}
	t.Fatalf("no %s query filtered on %q; queries: %+v", service, key, r.queries)
	return ""
}

func lineQty(c domain.Cost) map[string]float64 {
	out := map[string]float64{}
	for _, li := range c.LineItems {
		out[li.Dimension], _ = li.Quantity.Float64()
	}
	return out
}

func resource(kind string, attrs map[string]any) domain.Resource {
	return domain.Resource{Ref: domain.Reference{Kind: kind, Name: "x"}, Attributes: attrs}
}

func TestMSKPricesInstanceTypeAndPerBrokerStorage(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	c := estimateWith(t, rec, resource("aws_msk_cluster", map[string]any{
		"number_of_broker_nodes": 3.0,
		"broker_node_group_info": map[string]any{
			"instance_type": "kafka.m5.4xlarge",
			"storage_info": map[string]any{
				"ebs_storage_info": map[string]any{"volume_size": 1000.0},
			},
		},
	}))
	if got := rec.filter(t, "AmazonMSK", "computeFamily"); got != "m5.4xlarge" {
		t.Errorf("computeFamily = %q, want m5.4xlarge", got)
	}
	q := lineQty(c)
	if q["broker_hours"] != 3*730 || q["broker_storage"] != 3000 {
		t.Errorf("quantities = %v, want 2190 broker-hours and 3000 GB", q)
	}
}

func TestMSSQLDatabaseParsesSkuName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sku, license string
		product      string
		want         map[string]float64
	}{
		{
			"BC_Gen5_8", "", "SQL Database Single/Elastic Pool Business Critical - Compute Gen5",
			map[string]float64{"vcore_compute": 8 * 730, "vcore_license": 8 * 730, "storage": 32},
		},
		{
			"GP_Gen5_4", "BasePrice", "SQL Database Single/Elastic Pool General Purpose - Compute Gen5",
			map[string]float64{"vcore_compute": 4 * 730, "storage": 32},
		},
		{
			"HS_Gen5_2", "", "SQL Database SingleDB/Elastic Pool Hyperscale - Compute Gen5",
			map[string]float64{"vcore_compute": 2 * 730, "storage": 32},
		},
		{
			"GP_S_Gen5_2", "", "SQL Database General Purpose - Serverless - Compute Gen5",
			map[string]float64{"serverless_compute": 0.5 * 730, "storage": 32},
		},
		{
			"S0", "", "SQL Database Single Standard",
			map[string]float64{"dtu": 730.0 / 24},
		},
	}
	for _, tc := range cases {
		t.Run(tc.sku, func(t *testing.T) {
			t.Parallel()
			attrs := map[string]any{"sku_name": tc.sku}
			if tc.license != "" {
				attrs["license_type"] = tc.license
			}
			rec := &recorder{}
			c := estimateWith(t, rec, resource("azurerm_mssql_database", attrs))
			if got := rec.filter(t, "SQL Database", "productName"); got != tc.product {
				t.Errorf("productName = %q, want %q", got, tc.product)
			}
			got := lineQty(c)
			if len(got) != len(tc.want) {
				t.Fatalf("lines = %v, want %v", got, tc.want)
			}
			for id, q := range tc.want {
				if d := got[id] - q; d > 1e-9 || d < -1e-9 {
					t.Errorf("%s quantity = %v, want %v", id, got[id], q)
				}
			}
		})
	}
}

func TestAKSTierSelectsItsOwnMeter(t *testing.T) {
	t.Parallel()
	for tier, meter := range map[string]string{
		"Standard": "Standard Uptime SLA",
		"Premium":  "Standard Long Term Support",
	} {
		rec := &recorder{}
		estimateWith(t, rec, resource("azurerm_kubernetes_cluster", map[string]any{"sku_tier": tier}))
		if got := rec.filter(t, "Azure Kubernetes Service", "meterName"); got != meter {
			t.Errorf("%s: meterName = %q, want %q", tier, got, meter)
		}
	}
	rec := &recorder{}
	c := estimateWith(t, rec, resource("azurerm_kubernetes_cluster", map[string]any{"sku_tier": "Free"}))
	if len(c.LineItems) != 0 {
		t.Errorf("Free tier priced: %v", lineQty(c))
	}
}

func TestAuroraServerlessV2PricesACUs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		attrs      map[string]any
		filterKey  string
		filterWant string
		acuHours   float64
	}{
		{"standard at min capacity", map[string]any{
			"instance_class":                     "db.serverless",
			"serverlessv2_scaling_configuration": map[string]any{"min_capacity": 2.0, "max_capacity": 8.0},
		}, "usagetype", "Aurora:ServerlessV2Usage", 2 * 730},
		{"io-optimized, no scaling block", map[string]any{
			"instance_class": "db.serverless", "storage_type": "aurora-iopt1",
		}, "storage", "Aurora IO Optimization Mode", 0.5 * 730},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			c := estimateWith(t, rec, resource("aws_rds_cluster_instance", tc.attrs))
			if got := rec.filter(t, "AmazonRDS", tc.filterKey); got != tc.filterWant {
				t.Errorf("%s = %q, want %q", tc.filterKey, got, tc.filterWant)
			}
			q := lineQty(c)
			if len(q) != 1 || q["serverless_v2_acu"] != tc.acuHours {
				t.Errorf("lines = %v, want only serverless_v2_acu = %v", q, tc.acuHours)
			}
		})
	}
}

func TestCosmosThroughputScalesWithRegions(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	c := estimateWith(t, rec, resource("azurerm_cosmosdb_sql_container", map[string]any{
		"throughput": 400.0,
		"geo_location": []any{
			map[string]any{"location": "eastus"},
			map[string]any{"location": "westus"},
		},
		"multiple_write_locations_enabled": true,
	}))
	if got := rec.filter(t, "Azure Cosmos DB", "meterName"); got != "100 Multi-master RU/s" {
		t.Errorf("meterName = %q, want the multi-region write meter", got)
	}
	if q := lineQty(c); q["manual_throughput"] != 4*730*2 {
		t.Errorf("quantities = %v, want 5840 (4 x 730 x 2 regions)", q)
	}

	// No throughput of its own: a $0 line, not an unpriced resource.
	c = estimateWith(t, &recorder{}, resource("azurerm_cosmosdb_sql_container", map[string]any{}))
	if q := lineQty(c); len(q) != 1 || q["shared_throughput"] != 0 {
		t.Errorf("shared container lines = %v", q)
	}
}

func TestStorageAccountRedundancyAndTierDriveTheMeter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		attrs          map[string]any
		product, meter string
	}{
		{map[string]any{}, "Blob Storage", "Hot LRS Data Stored"},
		{map[string]any{"account_replication_type": "GRS"}, "Blob Storage", "Hot GRS Data Stored"},
		{map[string]any{"account_replication_type": "RAGRS", "access_tier": "Cool"}, "Blob Storage", "Cool RA-GRS Data Stored"},
		{map[string]any{"account_replication_type": "RAGZRS"}, "General Block Blob v2", "Hot RA-GZRS Data Stored"},
		{map[string]any{"account_replication_type": "GRS", "is_hns_enabled": true}, "General Block Blob v2 Hierarchical Namespace", "Hot GRS Data Stored"},
		{map[string]any{"account_tier": "Premium", "account_replication_type": "ZRS"}, "Premium Block Blob", "Premium ZRS Data Stored"},
	}
	for _, tc := range cases {
		rec := &recorder{}
		estimateWith(t, rec, resource("azurerm_storage_account", tc.attrs))
		if got := rec.filter(t, "Storage", "productName"); got != tc.product {
			t.Errorf("%v: productName = %q, want %q", tc.attrs, got, tc.product)
		}
		if got := rec.filter(t, "Storage", "meterName"); got != tc.meter {
			t.Errorf("%v: meterName = %q, want %q", tc.attrs, got, tc.meter)
		}
	}
}

func TestManagedDiskTierLadder(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind  string
		attrs map[string]any
		meter string
	}{
		// OS disks with no size take the image size: 30 GiB Linux, 127 GiB Windows.
		{"azurerm_linux_virtual_machine", map[string]any{"os_disk": map[string]any{"storage_account_type": "Premium_LRS"}}, "P4 LRS Disk"},
		{"azurerm_windows_virtual_machine", map[string]any{"os_disk": map[string]any{"storage_account_type": "Premium_LRS"}}, "P10 LRS Disk"},
		{"azurerm_linux_virtual_machine", map[string]any{"os_disk": map[string]any{"storage_account_type": "StandardSSD_ZRS", "disk_size_gb": 200.0}}, "E15 ZRS Disk"},
		{"azurerm_windows_virtual_machine", map[string]any{"os_disk": map[string]any{"storage_account_type": "Standard_LRS", "disk_size_gb": 8.0}}, "S4 LRS Disk"},
		{"azurerm_managed_disk", map[string]any{"storage_account_type": "Premium_LRS", "disk_size_gb": 512.0}, "P20 LRS Disk"},
		{"azurerm_managed_disk", map[string]any{"storage_account_type": "Premium_LRS", "disk_size_gb": 4096.0}, "P50 LRS Disk"},
	}
	for _, tc := range cases {
		rec := &recorder{}
		estimateWith(t, rec, resource(tc.kind, tc.attrs))
		if got := rec.filter(t, "Storage", "meterName"); got != tc.meter {
			t.Errorf("%s %v: meterName = %q, want %q", tc.kind, tc.attrs, got, tc.meter)
		}
	}
}

func TestCloudRunV2PricesMinInstances(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, &recorder{}, resource("google_cloud_run_v2_service", map[string]any{
		"template": map[string]any{
			"scaling": map[string]any{"min_instance_count": 2.0},
			"containers": map[string]any{
				"resources": map[string]any{
					"limits": map[string]any{"cpu": "500m", "memory": "1Gi"},
				},
			},
		},
	}))
	q := lineQty(c)
	const secs = 730 * 3600
	if q["min_instance_cpu"] != 2*0.5*secs || q["min_instance_memory"] != 2*1*secs {
		t.Errorf("quantities = %v, want %v vCPU-s and %v GiB-s", q, 2*0.5*secs, 2*secs)
	}
	for _, id := range []string{"vcpu", "memory", "requests"} {
		if _, ok := q[id]; !ok {
			t.Errorf("usage line %q missing; it must be emitted even without usage", id)
		}
	}
}
