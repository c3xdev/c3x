package plan_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/parser/plan"
)

// Shaped like `tofu show -json` output for a provider for_each: two
// instances of one resource on different provider instances, each with
// its region resolved in the plan, and a default provider in us-east-1.
const instancedPlan = `{
  "format_version": "1.2",
  "planned_values": {"root_module": {
    "resources": [
      {"address": "aws_instance.web[\"use1\"]", "mode": "managed", "type": "aws_instance", "name": "web", "index": "use1",
       "values": {"instance_type": "m5.large", "region": "us-east-1"}},
      {"address": "aws_instance.web[\"euw1\"]", "mode": "managed", "type": "aws_instance", "name": "web", "index": "euw1",
       "values": {"instance_type": "m5.large", "region": "eu-west-1"}},
      {"address": "aws_instance.legacy[0]", "mode": "managed", "type": "aws_instance", "name": "legacy", "index": 0,
       "values": {"instance_type": "t3.micro"}}
    ],
    "child_modules": [{"address": "module.app", "resources": [
      {"address": "module.app.aws_instance.web[\"a.b\"]", "mode": "managed", "type": "aws_instance", "name": "web", "index": "a.b",
       "values": {"instance_type": "m5.large"}}
    ]}]
  }},
  "configuration": {"provider_config": {"aws": {"name": "aws", "expressions": {"region": {"constant_value": "us-east-1"}}}}}
}`

func TestPlanKeepsInstanceKeysAndPerResourceRegion(t *testing.T) {
	t.Parallel()
	got, err := plan.ParseBytes([]byte(instancedPlan), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		`web["use1"]`:           "us-east-1",
		`web["euw1"]`:           "eu-west-1", // from the resource, not the default provider
		`legacy[0]`:             "us-east-1", // no region attribute: the default applies
		`module.app.web["a.b"]`: "us-east-1", // a dot inside the key must not confuse naming
	}
	if len(got) != len(want) {
		t.Fatalf("got %d resources, want %d", len(got), len(want))
	}
	for _, r := range got {
		region, ok := want[r.Ref.Name]
		if !ok {
			t.Errorf("unexpected name %q (instance keys must be kept so diffs pair instances correctly)", r.Ref.Name)
			continue
		}
		if r.Region == nil || *r.Region != region {
			t.Errorf("%s region = %v, want %s", r.Ref.Name, r.Region, region)
		}
	}
}
