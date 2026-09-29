package whatif_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/whatif"
)

func TestParseTerraformAddresses(t *testing.T) {
	cases := []struct {
		in         string
		kind, name string
		attr       string
		legacy     bool
	}{
		{`aws_instance.web.instance_type=m6i.large`, "aws_instance", "web", "instance_type", false},
		{`aws_instance.web[0].instance_type=m6i.large`, "aws_instance", "web[0]", "instance_type", false},
		{`module.tier["web"].aws_instance.this[0].instance_type=m6i.large`, "aws_instance", `module.tier["web"].this[0]`, "instance_type", false},
		{`module.a.module.b.aws_db_instance.main.instance_class=db.r6g.large`, "aws_db_instance", "module.a.module.b.main", "instance_class", false},
		{`aws_s3_bucket.logs["a.b"].versioning=true`, "aws_s3_bucket", `logs["a.b"]`, "versioning", false},
		// pre-0.3.19 form: still parsed to the same resource, flagged
		{`aws_instance.module.tier["web"].this[0].instance_type=m6i.large`, "aws_instance", `module.tier["web"].this[0]`, "instance_type", true},
	}
	for _, c := range cases {
		ovs, err := whatif.Parse([]string{c.in})
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		o := ovs[0]
		if o.Kind != c.kind || o.Name != c.name || o.Attr != c.attr || o.Legacy != c.legacy {
			t.Errorf("%s: got {%s %s %s legacy=%v}, want {%s %s %s legacy=%v}",
				c.in, o.Kind, o.Name, o.Attr, o.Legacy, c.kind, c.name, c.attr, c.legacy)
		}
	}
}

func TestParseRejectsIncompleteModuleAddress(t *testing.T) {
	if _, err := whatif.Parse([]string{`module.tier.aws_instance.instance_type=x`}); err == nil {
		t.Error("a module address without a resource name must be rejected")
	}
}
