package expr_test

import (
	"reflect"
	"testing"

	c3xexpr "github.com/c3xdev/c3x/internal/expr"
)

func TestIdentifiers(t *testing.T) {
	for src, want := range map[string][]string{
		`default(monthly_data_processed_gb, 0)`:                          {"monthly_data_processed_gb"},
		`monthly_hours()`:                                                nil,
		`default(instance_class, "db.t3.micro") == "x" ? price("a") : 0`: {"instance_class"},
		`root_block_device.volume_size * price("gb")`:                    {"root_block_device"},
		`max(default(min_size, 1), 2)`:                                   {"min_size"},
	} {
		got, err := c3xexpr.Identifiers(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", src, got, want)
		}
	}
}
