package cloudformation_test

import (
	"reflect"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/cloudformation"
)

// CloudFormation accepts strings for numeric and boolean properties, and
// every parameter value is a string. The parser hands the catalog typed
// values so its arithmetic and comparisons work.
func TestNumericAndBooleanStringsAreCoerced(t *testing.T) {
	t.Parallel()
	doc := `
AWSTemplateFormatVersion: "2010-09-09"
Parameters:
  Storage:
    Type: Number
    Default: "250"
Resources:
  DB:
    Type: AWS::RDS::DBInstance
    Properties:
      DBInstanceClass: db.r7g.large
      Engine: postgres
      EngineVersion: "8.0"
      AllocatedStorage: !Ref Storage
      Iops: "3000"
      MultiAZ: "true"
      DBInstanceIdentifier: "007"
`
	got, err := cloudformation.ParseBytes([]byte(doc), "t.yaml", cloudformation.Options{})
	if err != nil {
		t.Fatal(err)
	}
	attrs := got[0].Attributes
	want := map[string]any{
		"allocated_storage": 250.0,
		"Iops":              3000.0,
		"multi_az":          true,
		// Strings that would not read back identically keep their
		// spelling: a version like "8.0" or a zero-padded name.
		"EngineVersion": "8.0",
		"identifier":    "007",
		"engine":        "postgres",
	}
	for k, v := range want {
		if !reflect.DeepEqual(attrs[k], v) {
			t.Errorf("%s = %#v, want %#v", k, attrs[k], v)
		}
	}
}

func TestParameterOverrideStringsAreCoerced(t *testing.T) {
	t.Parallel()
	doc := `
Parameters:
  Size:
    Type: Number
    Default: 100
Resources:
  Vol:
    Type: AWS::EC2::Volume
    Properties:
      Size: !Ref Size
`
	got, err := cloudformation.ParseBytes([]byte(doc), "t.yaml", cloudformation.Options{
		Parameters: map[string]any{"Size": "500"}, // --var Size=500
	})
	if err != nil {
		t.Fatal(err)
	}
	if v := got[0].Attributes["size"]; v != 500.0 {
		t.Errorf("size = %#v, want 500", v)
	}
}

// BlockDeviceMappings become the root_block_device / ebs_block_device
// attributes the aws_instance catalog prices, the root chosen by the
// AMI's root device name.
func TestBlockDeviceMappingsMapToTerraformBlocks(t *testing.T) {
	t.Parallel()
	doc := `
Resources:
  Web:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: m7i.xlarge
      BlockDeviceMappings:
        - DeviceName: /dev/xvda
          Ebs:
            VolumeSize: 50
            VolumeType: gp3
        - DeviceName: /dev/xvdf
          Ebs:
            VolumeSize: "200"
            VolumeType: io2
            Iops: 4000
        - DeviceName: /dev/xvdg
          Ebs:
            VolumeSize: 100
        - DeviceName: /dev/sdb
          VirtualName: ephemeral0
        - DeviceName: /dev/xvdh
          NoDevice: {}
          Ebs:
            VolumeSize: 999
`
	got, err := cloudformation.ParseBytes([]byte(doc), "t.yaml", cloudformation.Options{})
	if err != nil {
		t.Fatal(err)
	}
	attrs := got[0].Attributes
	wantRoot := map[string]any{"volume_size": 50, "volume_type": "gp3"}
	if !reflect.DeepEqual(attrs["root_block_device"], wantRoot) {
		t.Errorf("root_block_device = %#v, want %#v", attrs["root_block_device"], wantRoot)
	}
	wantEBS := []any{
		map[string]any{"device_name": "/dev/xvdf", "volume_size": 200.0, "volume_type": "io2", "iops": 4000},
		map[string]any{"device_name": "/dev/xvdg", "volume_size": 100},
	}
	if !reflect.DeepEqual(attrs["ebs_block_device"], wantEBS) {
		t.Errorf("ebs_block_device = %#v, want %#v", attrs["ebs_block_device"], wantEBS)
	}
}

func TestLambdaArchitecturesAndEphemeralStorage(t *testing.T) {
	t.Parallel()
	doc := `
Resources:
  Fn:
    Type: AWS::Lambda::Function
    Properties:
      MemorySize: 1024
      Architectures: [arm64]
      EphemeralStorage:
        Size: 2048
`
	got, err := cloudformation.ParseBytes([]byte(doc), "t.yaml", cloudformation.Options{})
	if err != nil {
		t.Fatal(err)
	}
	attrs := got[0].Attributes
	if !reflect.DeepEqual(attrs["architectures"], []any{"arm64"}) {
		t.Errorf("architectures = %#v", attrs["architectures"])
	}
	if attrs["ephemeral_storage_size"] != 2048 {
		t.Errorf("ephemeral_storage_size = %#v", attrs["ephemeral_storage_size"])
	}
}

func TestIsTemplate(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"AWSTemplateFormatVersion: '2010-09-09'\nResources:\n  B:\n    Type: Custom::Thing\n":      true,
		"Resources:\n  B:\n    Type: AWS::S3::Bucket\n    Properties:\n      BucketName: !Ref N\n": true,
		`{"Resources":{"B":{"Type":"AWS::S3::Bucket"}}}`:                                           true,
		"version: 0.1\nresource_usage:\n  aws_s3_bucket.b:\n    storage_gb: 10\n":                  false,
		"Resources:\n  B:\n    Type: Custom::Thing\n":                                              false,
		"AWSTemplateFormatVersion: '2010-09-09'\nResources: {}\n":                                  false,
		"- a\n- b\n":                                     false,
		"apiVersion: v1\nkind: ConfigMap\n":              false,
		`{"format_version":"1.2","resource_changes":[]}`: false,
	}
	for doc, want := range cases {
		if got := cloudformation.IsTemplate([]byte(doc)); got != want {
			t.Errorf("IsTemplate(%q) = %v, want %v", doc, got, want)
		}
	}
}
