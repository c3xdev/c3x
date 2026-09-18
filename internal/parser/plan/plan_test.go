package plan_test

import (
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/plan"
)

func TestParsesPlanWithModuleAddresses(t *testing.T) {
	t.Parallel()

	raw := `{
		"resource_changes": [
			{
				"address": "module.frontend.aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["create"],
					"after": { "instance_type": "m5.large", "ami": "ami-123" }
				}
			},
			{
				"address": "aws_db_instance.main",
				"type": "aws_db_instance",
				"name": "main",
				"change": {
					"actions": ["create"],
					"after": { "instance_class": "db.t3.medium" }
				}
			},
			{
				"address": "aws_instance.gone",
				"type": "aws_instance",
				"name": "gone",
				"change": {
					"actions": ["delete"],
					"after": {}
				}
			}
		],
		"configuration": {
			"provider_config": {
				"aws": { "expressions": { "region": { "constant_value": "us-east-2" } } }
			}
		}
	}`

	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	// 2 created + 1 deleted (surfaced for delta renderer) = 3
	if len(got) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(got))
	}
	if got[0].Ref.Name != "module.frontend.web" {
		t.Errorf("module-prefixed name lost: %q", got[0].Ref.Name)
	}
	if got[1].Ref.Name != "main" {
		t.Errorf("top-level name = %q", got[1].Ref.Name)
	}
	if got[0].Region == nil || *got[0].Region != "us-east-2" {
		t.Errorf("region not picked up from provider_config: %v", got[0].Region)
	}
	if string(got[2].Action) != "delete" {
		t.Errorf("deleted resource action = %q, want 'delete'", got[2].Action)
	}
}

func TestPicksUpAzureLocation(t *testing.T) {
	t.Parallel()

	raw := `{
		"resource_changes": [
			{
				"address": "azurerm_storage_account.main",
				"type": "azurerm_storage_account",
				"name": "main",
				"change": {"actions": ["create"], "after": {}}
			}
		],
		"configuration": {
			"provider_config": {
				"azurerm": { "expressions": { "location": { "constant_value": "westeurope" } } }
			}
		}
	}`
	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Region == nil || *got[0].Region != "westeurope" {
		t.Errorf("expected westeurope, got %v", got[0].Region)
	}
}

func TestPicksUpGcpRegion(t *testing.T) {
	t.Parallel()

	raw := `{
		"resource_changes": [{
			"address": "google_compute_instance.x",
			"type": "google_compute_instance",
			"name": "x",
			"change": {"actions": ["create"], "after": {}}
		}],
		"configuration": {
			"provider_config": {
				"google": { "expressions": { "region": { "constant_value": "europe-west1" } } }
			}
		}
	}`
	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Region == nil || *got[0].Region != "europe-west1" {
		t.Errorf("expected europe-west1, got %v", got[0].Region)
	}
}

func TestRejectsInvalidJson(t *testing.T) {
	t.Parallel()
	_, err := plan.ParseBytes([]byte("not json"), nil)
	if err == nil {
		t.Fatalf("expected error on invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode plan") {
		t.Errorf("error message lacks context: %v", err)
	}
}

// TestPlannedValues_IncludesUnchangedResources is the core of the
// full-fleet decision (#51): estimating a plan prices every resource in
// the post-apply state, including unchanged (no-op) ones, not just the
// ones this plan changes.
func TestPlannedValues_IncludesUnchangedResources(t *testing.T) {
	t.Parallel()
	raw := `{
		"resource_changes": [
			{ "address": "aws_instance.new", "type": "aws_instance", "name": "new",
			  "change": { "actions": ["create"] } },
			{ "address": "aws_instance.existing", "type": "aws_instance", "name": "existing",
			  "change": { "actions": ["no-op"] } }
		],
		"planned_values": {
			"root_module": {
				"resources": [
					{ "address": "aws_instance.new", "mode": "managed", "type": "aws_instance", "name": "new", "values": { "instance_type": "t3.micro" } },
					{ "address": "aws_instance.existing", "mode": "managed", "type": "aws_instance", "name": "existing", "values": { "instance_type": "t3.large" } }
				]
			}
		}
	}`
	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resources (unchanged included), got %d", len(got))
	}
}

// TestPlannedValues_WalksChildModulesSafely covers the module tree walk:
// child modules are recursed, data sources are skipped, and a null
// child_modules element (untrusted plan JSON) does not panic.
func TestPlannedValues_WalksChildModulesSafely(t *testing.T) {
	t.Parallel()
	raw := `{
		"planned_values": {
			"root_module": {
				"resources": [
					{ "address": "data.aws_ami.x", "mode": "data", "type": "aws_ami", "name": "x", "values": {} }
				],
				"child_modules": [
					null,
					{ "address": "module.db",
					  "resources": [ { "address": "module.db.aws_db_instance.main", "mode": "managed", "type": "aws_db_instance", "name": "main", "values": { "instance_class": "db.t3.micro" } } ] }
				]
			}
		}
	}`
	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 managed resource (data + null child skipped), got %d", len(got))
	}
	if got[0].Ref.Kind != "aws_db_instance" {
		t.Errorf("expected the child-module db instance, got %s", got[0].Ref.Kind)
	}
}

// TestFallback_KeepsNoOpExcludesDeleteOnly checks the older-format path
// (no planned_values): unchanged resources are kept (they exist
// post-apply); resources scheduled only for destruction are surfaced
// with PlanActionDelete for the delta renderer.
func TestFallback_KeepsNoOpExcludesDeleteOnly(t *testing.T) {
	t.Parallel()
	raw := `{
		"resource_changes": [
			{ "address": "aws_instance.keep", "type": "aws_instance", "name": "keep",
			  "change": { "actions": ["no-op"], "after": { "instance_type": "t3.micro" } } },
			{ "address": "aws_instance.gone", "type": "aws_instance", "name": "gone",
			  "change": { "actions": ["delete"], "after": null } }
		]
	}`
	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	// 1 from fallback (no-op kept) + 1 appended delete = 2
	if len(got) != 2 {
		t.Fatalf("expected 2 resources (no-op + delete surfaced), got %d", len(got))
	}
	if got[0].Ref.Name != "keep" {
		t.Errorf("expected the no-op resource first, got %s", got[0].Ref.Name)
	}
	if string(got[1].Action) != "delete" {
		t.Errorf("expected delete action on second resource, got %q", got[1].Action)
	}
}

func TestPlannedValues_AnnotatesWithPlanAction(t *testing.T) {
	t.Parallel()

	raw := `{
		"resource_changes": [
			{
				"address": "aws_instance.created",
				"type": "aws_instance",
				"name": "created",
				"change": {"actions": ["create"], "after": {"instance_type": "t3.micro"}}
			},
			{
				"address": "aws_instance.updated",
				"type": "aws_instance",
				"name": "updated",
				"change": {"actions": ["update"], "after": {"instance_type": "t3.large"}}
			},
			{
				"address": "aws_instance.unchanged",
				"type": "aws_instance",
				"name": "unchanged",
				"change": {"actions": ["no-op"], "after": {"instance_type": "t3.small"}}
			}
		],
		"planned_values": {
			"root_module": {
				"resources": [
					{
						"address": "aws_instance.created",
						"mode": "managed",
						"type": "aws_instance",
						"name": "created",
						"values": {"instance_type": "t3.micro"}
					},
					{
						"address": "aws_instance.updated",
						"mode": "managed",
						"type": "aws_instance",
						"name": "updated",
						"values": {"instance_type": "t3.large"}
					},
					{
						"address": "aws_instance.unchanged",
						"mode": "managed",
						"type": "aws_instance",
						"name": "unchanged",
						"values": {"instance_type": "t3.small"}
					}
				]
			}
		}
	}`

	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(got))
	}

	actionByName := map[string]string{}
	for _, r := range got {
		actionByName[r.Ref.Name] = string(r.Action)
	}

	if actionByName["created"] != "create" {
		t.Errorf("created action = %q, want 'create'", actionByName["created"])
	}
	if actionByName["updated"] != "update" {
		t.Errorf("updated action = %q, want 'update'", actionByName["updated"])
	}
	if actionByName["unchanged"] != "no-op" {
		t.Errorf("unchanged action = %q, want 'no-op'", actionByName["unchanged"])
	}
}

func TestArrayExpressionsHandledGracefully(t *testing.T) {
	t.Parallel()

	// Terraform 4.x azurerm emits features as an array in expressions.
	// The parser must not crash on this.
	raw := `{
		"resource_changes": [
			{
				"address": "azurerm_resource_group.rg",
				"type": "azurerm_resource_group",
				"name": "rg",
				"change": {"actions": ["create"], "after": {"name": "my-rg", "location": "westeurope"}}
			}
		],
		"planned_values": {
			"root_module": {
				"resources": [
					{
						"address": "azurerm_resource_group.rg",
						"mode": "managed",
						"type": "azurerm_resource_group",
						"name": "rg",
						"values": {"name": "my-rg", "location": "westeurope"}
					}
				]
			}
		},
		"configuration": {
			"provider_config": {
				"azurerm": {
					"expressions": {
						"features": [{}],
						"location": {"constant_value": "westeurope"}
					}
				}
			}
		}
	}`

	got, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatalf("should not fail on array expressions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}
	if got[0].Region == nil || *got[0].Region != "westeurope" {
		t.Errorf("region should be westeurope despite array expression, got %v", got[0].Region)
	}
}

func TestParseBaseline_PricesBeforeState(t *testing.T) {
	// Plan with an update, a create, and a delete. The baseline (before)
	// set must contain only the resources that had prior state (the
	// update and the delete), each carrying its BEFORE attributes.
	raw := `{
		"format_version": "1.2",
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["update"],
					"before": { "instance_type": "m5.large" },
					"after":  { "instance_type": "m5.xlarge" }
				}
			},
			{
				"address": "aws_s3_bucket.data",
				"type": "aws_s3_bucket",
				"name": "data",
				"change": { "actions": ["create"], "before": null, "after": {} }
			},
			{
				"address": "aws_db_instance.old",
				"type": "aws_db_instance",
				"name": "old",
				"change": {
					"actions": ["delete"],
					"before": { "instance_class": "db.t3.medium" },
					"after": null
				}
			}
		]
	}`
	got, hasBaseline, err := plan.ParseBaselineBytes([]byte(raw), nil)
	if err != nil {
		t.Fatalf("ParseBaselineBytes: %v", err)
	}
	if !hasBaseline {
		t.Fatal("expected hasBaseline=true when resources have prior state")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 baseline resources (update + delete), got %d: %+v", len(got), got)
	}
	byName := map[string]map[string]any{}
	for _, r := range got {
		byName[r.Ref.Name] = r.Attributes
	}
	if byName["web"]["instance_type"] != "m5.large" {
		t.Errorf("baseline must use BEFORE attrs; got web=%v (want m5.large)", byName["web"])
	}
	if _, ok := byName["data"]; ok {
		t.Error("a create (before=null) must not appear in the baseline")
	}
	if _, ok := byName["old"]; !ok {
		t.Error("a delete must appear in the baseline (its before state)")
	}
}

func TestParseBaseline_GreenfieldHasNoBaseline(t *testing.T) {
	// Every resource is a create → no prior state → nothing to diff.
	raw := `{
		"resource_changes": [
			{ "address": "aws_instance.web", "type": "aws_instance", "name": "web",
			  "change": { "actions": ["create"], "before": null, "after": {} } }
		]
	}`
	got, hasBaseline, err := plan.ParseBaselineBytes([]byte(raw), nil)
	if err != nil {
		t.Fatalf("ParseBaselineBytes: %v", err)
	}
	if hasBaseline {
		t.Error("greenfield plan should report hasBaseline=false")
	}
	if len(got) != 0 {
		t.Errorf("greenfield baseline should be empty, got %d", len(got))
	}
}

func TestParsePostApply_ExcludesDeletions(t *testing.T) {
	// Update + create + delete (no planned_values, so the resource_changes
	// fallback is exercised). ParseBytes keeps the delete (for --show-delta);
	// ParsePostApplyBytes drops it (the plan-aware diff's "after" side).
	raw := `{
		"resource_changes": [
			{ "address": "aws_instance.web", "type": "aws_instance", "name": "web",
			  "change": { "actions": ["update"], "before": {"instance_type":"m5.large"}, "after": {"instance_type":"m5.xlarge"} } },
			{ "address": "aws_s3_bucket.data", "type": "aws_s3_bucket", "name": "data",
			  "change": { "actions": ["create"], "before": null, "after": {} } },
			{ "address": "aws_db_instance.old", "type": "aws_db_instance", "name": "old",
			  "change": { "actions": ["delete"], "before": {"instance_class":"db.t3.medium"}, "after": null } }
		]
	}`
	names := func(rs []domain.Resource) map[string]bool {
		m := map[string]bool{}
		for _, r := range rs {
			m[r.Ref.Name] = true
		}
		return m
	}

	full, err := plan.ParseBytes([]byte(raw), nil)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	if !names(full)["old"] {
		t.Error("ParseBytes must keep the delete-only resource (for --show-delta)")
	}

	post, err := plan.ParsePostApplyBytes([]byte(raw), nil)
	if err != nil {
		t.Fatalf("ParsePostApplyBytes: %v", err)
	}
	pn := names(post)
	if pn["old"] {
		t.Error("ParsePostApplyBytes must EXCLUDE the delete-only resource (#62)")
	}
	if !pn["web"] || !pn["data"] {
		t.Errorf("ParsePostApplyBytes should keep the update and the create; got %v", pn)
	}
	if len(post) != 2 {
		t.Errorf("post-apply set should be 2 (web, data), got %d", len(post))
	}
}
