package calculator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser/cloudformation"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

// byFilter prices a lookup by the value one attribute filter carries, so a
// test can tell which of two mappings a rate expression chose.
type byFilter struct {
	key   string
	rates map[string]string
}

func (b byFilter) Lookup(_ context.Context, q pricing.Query) (decimal.Decimal, string, error) {
	for _, kv := range q.AttributeFilters {
		if kv.Key == b.key {
			if r, ok := b.rates[kv.Value]; ok {
				return decimal.RequireFromString(r), domain.PriceSourceLive, nil
			}
		}
	}
	return decimal.NewFromInt(1), domain.PriceSourceLive, nil
}

func lineRate(c domain.Cost, dim string) string {
	for _, li := range c.LineItems {
		if li.Dimension == dim {
			return li.UnitRate.String()
		}
	}
	return ""
}

// architectures is a list (["arm64"]); an arm64 function is billed at the
// Graviton duration rate and an x86 one (or one with no architectures)
// at the x86 rate.
func TestLambdaArchitecturePicksDurationRate(t *testing.T) {
	t.Parallel()
	src := byFilter{key: "group", rates: map[string]string{
		"AWS-Lambda-Duration":     "0.0000166667",
		"AWS-Lambda-Duration-ARM": "0.0000133334",
	}}
	cases := map[string]struct {
		arch any
		want string
	}{
		"arm64 list":  {[]any{"arm64"}, "0.0000133334"},
		"x86_64 list": {[]any{"x86_64"}, "0.0000166667"},
		"unset":       {nil, "0.0000166667"},
		"arm64 bare":  {"arm64", "0.0000133334"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attrs := map[string]any{"memory_size": 1024.0, "monthly_requests": 1e6}
			if tc.arch != nil {
				attrs["architectures"] = tc.arch
			}
			c := estimateWith(t, src, resource("aws_lambda_function", attrs))
			if got := lineRate(c, "duration"); got != tc.want {
				t.Errorf("duration rate = %s, want %s", got, tc.want)
			}
		})
	}
}

// The VM size is Azure's armSkuName; skuName is "B2s" / "B2s v2" for the
// B-series, so filtering on it found no price.
func TestAzureVMsFilterOnArmSkuName(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{
		"azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine",
	} {
		rec := &recorder{}
		estimateWith(t, rec, resource(kind, map[string]any{"size": "Standard_B2s_v2", "location": "westeurope"}))
		if got := rec.filter(t, "Virtual Machines", "armSkuName"); got != "Standard_B2s_v2" {
			t.Errorf("%s: armSkuName = %q, want Standard_B2s_v2", kind, got)
		}
		for _, q := range rec.queries {
			for _, kv := range q.AttributeFilters {
				if q.Service == "Virtual Machines" && kv.Key == "skuName" {
					t.Errorf("%s still filters on skuName = %q", kind, kv.Value)
				}
			}
		}
	}
}

// Cloud SQL SKUs are named after the location they are sold in, which is
// per region ("vCPU in Frankfurt"), not "in Americas" everywhere. The
// location follows the resource's region attribute, and the region it
// is priced in when the attribute is unset.
func TestCloudSQLDescriptionFollowsRegion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		attrs  map[string]any
		region *string
		want   string
	}{
		{
			"attribute",
			map[string]any{"region": "europe-west3"},
			nil,
			"Cloud SQL for PostgreSQL: Zonal - vCPU in Frankfurt",
		},
		{
			"resource region",
			map[string]any{},
			region("asia-southeast1"),
			"Cloud SQL for PostgreSQL: Zonal - vCPU in Singapore",
		},
		{
			"americas",
			map[string]any{"region": "us-east1"},
			nil,
			"Cloud SQL for PostgreSQL: Zonal - vCPU in Americas",
		},
		{
			"unknown region",
			map[string]any{"region": "mars-north1"},
			nil,
			"Cloud SQL for PostgreSQL: Zonal - vCPU in Americas",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.attrs["tier"] = "db-custom-2-7680"
			r := resource("google_sql_database_instance", tc.attrs)
			r.Region = tc.region
			rec := &recorder{}
			estimateWith(t, rec, r)
			var got string
			for _, q := range rec.queries {
				for _, kv := range q.AttributeFilters {
					if kv.Key == "description" && strings.Contains(kv.Value, "vCPU") {
						got = kv.Value
					}
				}
			}
			if got != tc.want {
				t.Errorf("vCPU description = %q, want %q", got, tc.want)
			}
		})
	}
}

// A CloudFormation template writing numbers as strings and declaring EBS
// volumes in BlockDeviceMappings estimates end to end: the storage is
// 250 GB (not an "expected number, got string" error) and the root and
// extra volumes are priced.
func TestCloudFormationStringNumbersAndBlockDevices(t *testing.T) {
	t.Parallel()
	doc := `
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  App:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: m7i.xlarge
      BlockDeviceMappings:
        - DeviceName: /dev/xvda
          Ebs: {VolumeSize: 50, VolumeType: gp3}
        - DeviceName: /dev/xvdf
          Ebs: {VolumeSize: "200", VolumeType: gp3}
  DB:
    Type: AWS::RDS::DBInstance
    Properties:
      DBInstanceClass: db.r7g.large
      Engine: postgres
      AllocatedStorage: "250"
      StorageType: gp3
      MultiAZ: "false"
`
	res, err := cloudformation.ParseBytes([]byte(doc), "t.yaml", cloudformation.Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		q := lineQty(estimateWith(t, &recorder{}, r))
		switch r.Ref.Kind {
		case "aws_instance":
			if q["root_volume"] != 50 || q["ebs_volumes"] != 200 {
				t.Errorf("instance volumes = %v, want root 50 GB and 200 GB extra", q)
			}
		case "aws_db_instance":
			if q["storage"] != 250 {
				t.Errorf("db storage = %v, want 250 GB", q)
			}
		}
	}
}
