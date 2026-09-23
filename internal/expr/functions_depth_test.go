package expr_test

// Tests for the depth-support helpers: two-level block flattening,
// sum_field, count_blocks.

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/expr"
	"github.com/shopspring/decimal"
)

func evalQty(t *testing.T, src string, attrs map[string]any) float64 {
	t.Helper()
	r := domain.Resource{Ref: domain.Reference{Kind: "k", Name: "n"}, Attributes: attrs}
	lookup := func(string) (decimal.Decimal, string, error) {
		return decimal.Zero, "", nil
	}
	prog, err := expr.Compile(src)
	if err != nil {
		t.Fatalf("Compile(%q): %v", src, err)
	}
	got, err := expr.RunNumber(prog, expr.EnvFor(r, lookup, nil))
	if err != nil {
		t.Fatalf("RunNumber(%q): %v", src, err)
	}
	return got
}

func TestTwoLevelFlattening(t *testing.T) {
	t.Parallel()
	got := evalQty(t, "default(boot_disk_initialize_params_size, 10)", map[string]any{
		"boot_disk": map[string]any{
			"initialize_params": map[string]any{"size": 100},
		},
	})
	if got != 100 {
		t.Errorf("two-level flatten = %v, want 100", got)
	}
}

func TestSumFieldAcrossRepeatedBlocks(t *testing.T) {
	t.Parallel()
	attrs := map[string]any{
		"ebs_block_device": []any{
			map[string]any{"volume_size": 100},
			map[string]any{"volume_size": 250},
			map[string]any{"no_size_here": true},
		},
	}
	if got := evalQty(t, `sum_field(ebs_block_device, "volume_size")`, attrs); got != 350 {
		t.Errorf("sum_field = %v, want 350", got)
	}
	if got := evalQty(t, `sum_field(missing_attr, "volume_size")`, attrs); got != 0 {
		t.Errorf("sum_field(nil) = %v, want 0", got)
	}
	if got := evalQty(t, `count_blocks(ebs_block_device)`, attrs); got != 3 {
		t.Errorf("count_blocks = %v, want 3", got)
	}
}
