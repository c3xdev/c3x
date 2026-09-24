package parser

import "testing"

func TestRegionFromAttributes(t *testing.T) {
	cases := []struct {
		kind  string
		attrs map[string]any
		want  string
	}{
		{"azurerm_linux_virtual_machine", map[string]any{"location": "westeurope"}, "westeurope"},
		{"azurerm_storage_account", map[string]any{"location": "West Europe"}, "westeurope"},
		{"azurerm_linux_virtual_machine", map[string]any{"location": nil}, ""}, // unresolved
		{"google_compute_instance", map[string]any{"zone": "europe-west4-b"}, "europe-west4"},
		{"google_sql_database_instance", map[string]any{"region": "asia-east1"}, "asia-east1"},
		{"google_container_cluster", map[string]any{"location": "us-central1-a"}, "us-central1"},
		{"google_container_cluster", map[string]any{"location": "europe-west1"}, "europe-west1"},
		{"google_storage_bucket", map[string]any{"location": "US"}, ""}, // multi-region, not a region
		{"aws_instance", map[string]any{"region": "eu-west-1"}, "eu-west-1"},
		{"aws_instance", map[string]any{}, ""},
	}
	for _, c := range cases {
		if got := regionFromAttributes(c.kind, c.attrs); got != c.want {
			t.Errorf("%s %v: got %q, want %q", c.kind, c.attrs, got, c.want)
		}
	}
}
