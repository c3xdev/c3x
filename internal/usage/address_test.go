package usage_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/usage"
)

func moduleResource() []domain.Resource {
	return []domain.Resource{{Ref: domain.Reference{Kind: "aws_instance", Name: `module.tier["web"].this[0]`}}}
}

func TestUsageKeyTerraformAddress(t *testing.T) {
	rs := moduleResource()
	rep := usage.ApplyWithReport(rs, usage.File{ResourceUsage: map[string]map[string]any{
		`module.tier["web"].aws_instance.this[0]`: {"monthly_hours": 100},
	}})
	if len(rep.Unmatched) != 0 || len(rep.Legacy) != 0 {
		t.Fatalf("report = %+v, want a clean match", rep)
	}
	if rs[0].Attributes["monthly_hours"] != 100 {
		t.Errorf("usage not applied: %v", rs[0].Attributes)
	}
}

// Keys written in the pre-0.3.19 form still match, and are reported so
// the CLI can suggest the rename.
func TestUsageKeyLegacyFormStillMatches(t *testing.T) {
	rs := moduleResource()
	rep := usage.ApplyWithReport(rs, usage.File{ResourceUsage: map[string]map[string]any{
		`aws_instance.module.tier["web"].this[0]`: {"monthly_hours": 50},
	}})
	if len(rep.Unmatched) != 0 {
		t.Fatalf("legacy key unmatched: %v", rep.Unmatched)
	}
	if rs[0].Attributes["monthly_hours"] != 50 {
		t.Errorf("legacy key not applied: %v", rs[0].Attributes)
	}
	want := `module.tier["web"].aws_instance.this[0]`
	if got := rep.Legacy[`aws_instance.module.tier["web"].this[0]`]; got != want {
		t.Errorf("Legacy suggestion = %q, want %q", got, want)
	}
}

// With both forms in one file, the Terraform address wins and the legacy
// key is reported as unmatched rather than silently merged.
func TestUsageKeyTerraformAddressWinsOverLegacy(t *testing.T) {
	rs := moduleResource()
	rep := usage.ApplyWithReport(rs, usage.File{ResourceUsage: map[string]map[string]any{
		`module.tier["web"].aws_instance.this[0]`: {"monthly_hours": 100},
		`aws_instance.module.tier["web"].this[0]`: {"monthly_hours": 50},
	}})
	if rs[0].Attributes["monthly_hours"] != 100 {
		t.Errorf("monthly_hours = %v, want the Terraform-address value", rs[0].Attributes["monthly_hours"])
	}
	if len(rep.Unmatched) != 1 || rep.Unmatched[0] != `aws_instance.module.tier["web"].this[0]` {
		t.Errorf("unmatched = %v, want the shadowed legacy key", rep.Unmatched)
	}
}
