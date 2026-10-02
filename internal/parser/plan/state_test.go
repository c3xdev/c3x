package plan_test

import (
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/plan"
)

const stateJSON = `{
	"format_version": "1.0",
	"terraform_version": "1.9.0",
	"values": {
		"root_module": {
			"resources": [
				{
					"address": "aws_s3_bucket.data",
					"mode": "managed",
					"type": "aws_s3_bucket",
					"name": "data",
					"values": { "bucket": "acme-data", "region": "eu-west-1" }
				},
				{
					"address": "data.aws_caller_identity.me",
					"mode": "data",
					"type": "aws_caller_identity",
					"name": "me",
					"values": { "account_id": "123456789012" }
				}
			],
			"child_modules": [
				{
					"address": "module.m",
					"resources": [
						{
							"address": "module.m.aws_s3_bucket.x[\"a\"]",
							"mode": "managed",
							"type": "aws_s3_bucket",
							"name": "x",
							"index": "a",
							"values": { "bucket": "acme-x-a" }
						}
					],
					"child_modules": [null]
				}
			]
		}
	}
}`

func TestParseState_ReadsResourcesAcrossModules(t *testing.T) {
	t.Parallel()

	got, err := plan.ParseStateBytes([]byte(stateJSON), nil)
	if err != nil {
		t.Fatalf("ParseStateBytes: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 managed resources (data source skipped), got %d: %+v", len(got), got)
	}

	root, nested := got[0], got[1]
	if root.Ref.Label() != "aws_s3_bucket.data" {
		t.Errorf("root label = %q", root.Ref.Label())
	}
	if root.Attributes["bucket"] != "acme-data" {
		t.Errorf("root bucket = %v, want the real name from state", root.Attributes["bucket"])
	}
	if root.Region == nil || *root.Region != "eu-west-1" {
		t.Errorf("root region = %v, want eu-west-1 from values.region", root.Region)
	}

	// The label must equal the state address: it is the key a usage file
	// is looked up by.
	if want := `module.m.aws_s3_bucket.x["a"]`; nested.Ref.Label() != want {
		t.Errorf("nested label = %q, want %q", nested.Ref.Label(), want)
	}
	if nested.Region != nil {
		t.Errorf("nested region = %v, want nil when state carries none", *nested.Region)
	}
}

func TestParseState_EmptyStateIsNotAnError(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`{"format_version":"1.0"}`, `{"values":{}}`, `{"values":{"root_module":{}}}`} {
		got, err := plan.ParseStateBytes([]byte(raw), nil)
		if err != nil || len(got) != 0 {
			t.Errorf("%s: got %d resources, err %v; want none and no error", raw, len(got), err)
		}
	}
}

func TestParseState_RejectsAPlan(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"planned_values":   `{"planned_values":{"root_module":{}}}`,
		"resource_changes": `{"resource_changes":[{"address":"a.b","type":"a","name":"b","change":{"actions":["create"]}}]}`,
	} {
		_, err := plan.ParseStateBytes([]byte(raw), nil)
		if err == nil || !strings.Contains(err.Error(), "a plan, not a state") {
			t.Errorf("%s: err = %v, want a plan-not-state error", name, err)
		}
	}
}

func TestParseState_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := plan.ParseStateBytes([]byte("{not json"), nil); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
