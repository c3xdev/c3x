// Package cloudformation parses AWS CloudFormation templates into the
// same [domain.Resource] type the Terraform parser emits. Downstream
// the calculator can't tell the two apart — the catalog is keyed on
// Terraform resource kinds, so CloudFormation resources get
// translated into their Terraform equivalents on the way through.
//
// The translation tables in this file are the contract: a CFN type
// like `AWS::EC2::Instance` becomes a Terraform-shaped resource of
// kind `aws_instance` with attribute names rewritten to the names
// the catalog expressions reference (e.g. `InstanceType` →
// `instance_type`).
package cloudformation

// typeMap is the CFN → Terraform resource-kind lookup. Entries cover
// the most common cost-relevant resource types; the long tail is
// reachable by extending this map plus the corresponding property
// translator.
//
// Coverage targets the catalog's AWS surface — adding a CFN type
// whose Terraform equivalent isn't in the catalog yields a zero-cost
// row (same behaviour the Terraform parser exhibits for unknown
// kinds).
var typeMap = map[string]string{
	"AWS::EC2::Instance":                        "aws_instance",
	"AWS::EC2::Volume":                          "aws_ebs_volume",
	"AWS::EC2::EIP":                             "aws_eip",
	"AWS::EC2::NatGateway":                      "aws_nat_gateway",
	"AWS::EC2::VPCEndpoint":                     "aws_vpc_endpoint",
	"AWS::ElasticLoadBalancingV2::LoadBalancer": "aws_lb",
	"AWS::ElasticLoadBalancingV2::TargetGroup":  "aws_lb_target_group",
	"AWS::S3::Bucket":                           "aws_s3_bucket",
	"AWS::RDS::DBInstance":                      "aws_db_instance",
	"AWS::RDS::DBCluster":                       "aws_rds_cluster",
	"AWS::DynamoDB::Table":                      "aws_dynamodb_table",
	"AWS::ElastiCache::CacheCluster":            "aws_elasticache_cluster",
	"AWS::Lambda::Function":                     "aws_lambda_function",
	"AWS::EKS::Cluster":                         "aws_eks_cluster",
	"AWS::StepFunctions::StateMachine":          "aws_sfn_state_machine",
	"AWS::CloudFront::Distribution":             "aws_cloudfront_distribution",
	"AWS::Route53::HostedZone":                  "aws_route53_zone",
	"AWS::ApiGateway::RestApi":                  "aws_api_gateway_rest_api",
	"AWS::ApiGatewayV2::Api":                    "aws_apigatewayv2_api",
	"AWS::Kinesis::Stream":                      "aws_kinesis_stream",
	"AWS::KMS::Key":                             "aws_kms_key",
	"AWS::SecretsManager::Secret":               "aws_secretsmanager_secret",
	"AWS::SQS::Queue":                           "aws_sqs_queue",
	"AWS::SNS::Topic":                           "aws_sns_topic",
	"AWS::Logs::LogGroup":                       "aws_cloudwatch_log_group",
	"AWS::WAFv2::WebACL":                        "aws_wafv2_web_acl",
	"AWS::ECR::Repository":                      "aws_ecr_repository",
	"AWS::DocDB::DBCluster":                     "aws_documentdb_cluster",
	"AWS::Neptune::DBCluster":                   "aws_neptune_cluster",
	"AWS::Redshift::Cluster":                    "aws_redshift_cluster",
	"AWS::OpenSearchService::Domain":            "aws_opensearch_domain",
	"AWS::EFS::FileSystem":                      "aws_efs_file_system",
	"AWS::CertificateManager::Certificate":      "aws_acm_certificate",
}

// propertyMap rewrites CFN property names to the lower-snake-case
// keys the catalog expressions read. We rewrite only the
// cost-relevant subset; properties the catalog doesn't reference
// pass through under their CFN name (which the calculator's
// `default(...)` falls back to silently).
//
// Each map is keyed by Terraform resource kind (after translation)
// so adding a new CFN type means adding one entry per kind, not one
// per CFN-property.
var propertyMap = map[string]map[string]string{
	"aws_instance": {
		"InstanceType": "instance_type",
		"ImageId":      "ami",
		// BlockDeviceMappings is a list of objects rather than a
		// scalar; translateBlockDevices maps it onto the Terraform
		// root_block_device / ebs_block_device shape.
	},
	"aws_ebs_volume": {
		"Size":       "size",
		"VolumeType": "type",
		"Iops":       "iops",
		"Throughput": "throughput",
	},
	"aws_lb": {
		"Type": "load_balancer_type",
	},
	"aws_s3_bucket": {
		"BucketName": "bucket",
	},
	"aws_db_instance": {
		"DBInstanceClass":      "instance_class",
		"Engine":               "engine",
		"AllocatedStorage":     "allocated_storage",
		"StorageType":          "storage_type",
		"MultiAZ":              "multi_az",
		"DBInstanceIdentifier": "identifier",
	},
	"aws_rds_cluster": {
		"Engine": "engine",
	},
	"aws_dynamodb_table": {
		"BillingMode": "billing_mode",
		"TableName":   "name",
	},
	"aws_elasticache_cluster": {
		"CacheNodeType": "node_type",
		"Engine":        "engine",
		"NumCacheNodes": "num_cache_nodes",
	},
	"aws_lambda_function": {
		"MemorySize":            "memory_size",
		"FunctionName":          "function_name",
		"Runtime":               "runtime",
		"Architectures":         "architectures",
		"EphemeralStorage.Size": "ephemeral_storage_size",
	},
	"aws_redshift_cluster": {
		"NodeType":      "node_type",
		"NumberOfNodes": "number_of_nodes",
	},
	"aws_opensearch_domain": {
		// OpenSearch shoves config into a deeply-nested ClusterConfig
		// property; we flatten the leaves the catalog reads.
		"ClusterConfig.InstanceType":  "cluster_config_instance_type",
		"ClusterConfig.InstanceCount": "cluster_config_instance_count",
		"EBSOptions.EBSEnabled":       "ebs_options_ebs_enabled",
		"EBSOptions.VolumeType":       "ebs_options_volume_type",
		"EBSOptions.VolumeSize":       "ebs_options_volume_size",
	},
}

// translateKind looks up the Terraform-equivalent kind for a CFN
// type. Returns "" for unknown types so the parser can skip them
// with a logged warning rather than emitting bogus rows.
func translateKind(cfnType string) string { return typeMap[cfnType] }

// translateProps rewrites a property map from CFN names to the
// snake_case keys the catalog reads. Unknown property keys carry
// through unchanged so authors of new resource types don't have to
// update both this file and the catalog to land a value.
func translateProps(kind string, cfn map[string]any) map[string]any {
	rewrites, ok := propertyMap[kind]
	if !ok {
		// No translation registered: pass props through verbatim.
		return cfn
	}
	out := make(map[string]any, len(cfn))
	flat := flattenNested(cfn, "")
	for cfnKey, val := range flat {
		if tfKey, ok := rewrites[cfnKey]; ok {
			out[tfKey] = val
			continue
		}
		// Pass through the unmodified leaf so the catalog's default()
		// can still pick up an attribute we didn't formally rewrite.
		out[cfnKey] = val
	}
	if kind == "aws_instance" {
		translateBlockDevices(out)
	}
	return out
}

// rootDeviceNames are the device names EC2 AMIs use for the root
// volume: /dev/xvda on Amazon Linux and most HVM images, /dev/sda1 on
// Ubuntu, Windows and older AMIs.
var rootDeviceNames = map[string]bool{
	"/dev/xvda": true, "xvda": true,
	"/dev/sda1": true, "sda1": true,
	"/dev/sda": true, "sda": true,
	"/dev/nvme0n1": true,
}

// translateBlockDevices rewrites an instance's CloudFormation
// BlockDeviceMappings into the Terraform aws_instance shape the catalog
// prices: the mapping on the AMI's root device name becomes
// `root_block_device`, every other EBS mapping an `ebs_block_device`
// entry. Instance-store (VirtualName) and suppressed (NoDevice)
// mappings carry no EBS volume and are skipped.
func translateBlockDevices(attrs map[string]any) {
	mappings, ok := attrs["BlockDeviceMappings"].([]any)
	if !ok {
		return
	}
	var extra []any
	for _, m := range mappings {
		bdm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		ebs, ok := bdm["Ebs"].(map[string]any)
		if !ok {
			continue
		}
		if _, suppressed := bdm["NoDevice"]; suppressed {
			continue
		}
		device, _ := bdm["DeviceName"].(string)
		vol := map[string]any{}
		if device != "" {
			vol["device_name"] = device
		}
		for cfnKey, tfKey := range map[string]string{
			"VolumeSize": "volume_size",
			"VolumeType": "volume_type",
			"Iops":       "iops",
			"Throughput": "throughput",
		} {
			if v, ok := ebs[cfnKey]; ok && v != nil {
				vol[tfKey] = v
			}
		}
		_, haveRoot := attrs["root_block_device"]
		if rootDeviceNames[device] && !haveRoot {
			delete(vol, "device_name")
			attrs["root_block_device"] = vol
			continue
		}
		extra = append(extra, vol)
	}
	if len(extra) > 0 {
		attrs["ebs_block_device"] = extra
	}
}

// flattenNested expands `{ "ClusterConfig": { "InstanceType": "x" } }`
// into `{ "ClusterConfig.InstanceType": "x" }` so the dot-keyed
// propertyMap entries (like OpenSearch's `ClusterConfig.InstanceType`)
// match without callers having to remember the structure.
func flattenNested(in map[string]any, prefix string) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			for fk, fv := range flattenNested(t, key) {
				out[fk] = fv
			}
		default:
			out[key] = v
		}
	}
	return out
}
