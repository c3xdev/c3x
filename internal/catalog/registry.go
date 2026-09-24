package catalog

import (
	"fmt"
	"sort"
)

// Registry is the indexed set of catalog Definitions keyed by Kind.
// Construct via Load or LoadFromFS; do not instantiate directly so the
// validation guarantees hold.
type Registry struct {
	byKind map[string]*Definition
}

func newRegistry() *Registry { return &Registry{byKind: map[string]*Definition{}} }

func (r *Registry) add(def *Definition) error {
	if existing, ok := r.byKind[def.Kind]; ok {
		return fmt.Errorf("duplicate definition for kind %q (first from %q)",
			def.Kind, existing.DisplayName)
	}
	r.byKind[def.Kind] = def
	return nil
}

// Get returns the Definition for the given Kind, or nil if unknown.
// Calculator callers treat a nil return as "we don't price this kind"
// and surface a zero-cost row rather than failing the whole estimate.
func (r *Registry) Get(kind string) *Definition { return r.byKind[kind] }

// Len reports the number of registered definitions.
func (r *Registry) Len() int { return len(r.byKind) }

// Kinds returns every registered Kind in sorted order. Used by the
// `c3x resources` command and the verifier harness.
func (r *Registry) Kinds() []string {
	out := make([]string, 0, len(r.byKind))
	for k := range r.byKind {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LegitimatelyFreeKinds enumerates resources whose zero-cost
// breakdown is correct rather than a regression. Each entry is
// either always-free (public ACM certificates) or rolled into a
// parent resource (ALB target groups bill via aws_lb). The verifier
// reports these as `[FREE]` rather than `[ZERO]` so it can hard-fail
// CI on real regressions without flagging known-good zeros.
//
// Adding to this set is a deliberate catalog-level decision; the
// verifier intentionally doesn't accept overrides from CLI flags so
// a hot-fix can't sneak a quietly-broken resource through as "free".
var LegitimatelyFreeKinds = map[string]string{
	"aws_acm_certificate":  "public ACM certificates are issued and renewed for free",
	"aws_lb_target_group":  "billed via the parent aws_lb",
	"aws_efs_access_point": "free addressing construct on top of aws_efs_file_system",
	"aws_efs_mount_target": "free per-AZ endpoint to access an EFS file system",
	"aws_security_group":   "AWS does not charge for security groups",
	"aws_internet_gateway": "no per-resource charge (data transfer billed elsewhere)",
	"aws_iam_role":         "IAM is a free service",

	// Glue Catalog Database itself is free (first 1M objects);
	// stored objects can bill but that's typed as a STATIC dimension
	// gated by `when`.
	"aws_glue_catalog_database":               "Glue Data Catalog Database — first 1M objects free; usage modelled as a `when`-gated dimension",
	"aws_directconnect_lag":                   "Direct Connect Link Aggregation Group — no per-resource charge; ports bill via aws_dx_connection",
	"aws_directconnect_gateway":               "Direct Connect Gateway — no per-resource charge; data transfer rides on aws_dx_connection",
	"aws_lambda_function_url":                 "free (function invocations bill via aws_lambda_function)",
	"aws_chatbot_slack_channel_configuration": "free (AWS Chatbot is a no-charge service)",

	// VPC plumbing — all free; data transfer + per-resource fees
	// live on the resources that actually use them.
	"aws_subnet":                       "VPC subnet — no per-resource charge",
	"aws_vpc":                          "VPC itself is free; resources inside it bill",
	"aws_route_table":                  "free",
	"aws_route":                        "free",
	"aws_main_route_table_association": "free",
	"aws_vpn_gateway":                  "free (data plane hours bill on aws_vpn_connection)",
	"aws_customer_gateway":             "free",
	"aws_vpc_peering_connection":       "free (cross-region data transfer bills on the originating resource)",
	"aws_eip_association":              "free (the EIP itself is billed via aws_eip)",
	"aws_network_interface":            "free unless detached (AWS charges for unattached ENIs via aws_eip)",

	// IAM — entirely free service.
	"aws_iam_policy":                 "IAM is free",
	"aws_iam_user":                   "IAM is free",
	"aws_iam_group":                  "IAM is free",
	"aws_iam_role_policy":            "IAM is free",
	"aws_iam_role_policy_attachment": "IAM is free",

	// API Gateway — free metadata; API calls bill via aws_api_gateway_rest_api.
	"aws_apigateway_account":  "API Gateway account settings — no charge",
	"aws_apigateway_resource": "free (REST resources are metadata)",
	"aws_apigateway_method":   "free (priced via parent aws_api_gateway_rest_api)",

	// Auto scaling / launch metadata — no per-resource charge.
	"aws_appautoscaling_target": "Application Auto Scaling is free",
	"aws_appautoscaling_policy": "free",
	"aws_autoscaling_group":     "free (billed via EC2 instances it launches)",
	"aws_launch_template":       "free",
	"aws_launch_configuration":  "free",

	// ECS / EKS control-plane wiring — free until you attach data.
	"aws_ecs_cluster":         "free (Fargate / EC2 tasks bill separately)",
	"aws_ecs_task_definition": "free (definitions are metadata)",
	// aws_ecs_service is NOT in this list: FARGATE launch type bills
	// vCPU/GB-hours (modelled in its TOML); only EC2 launch type is
	// free at the service level.
	// aws_eks_node_group is NOT in this list: it prices the EC2
	// instances its scaling_config launches.
	"aws_eks_fargate_profile": "free (billed via Fargate task hours)",

	// KMS / messaging wiring.
	"aws_kms_alias":                     "free (the underlying aws_kms_key bills)",
	"aws_kms_grant":                     "free",
	"aws_sqs_queue_policy":              "free (queue itself bills via aws_sqs_queue)",
	"aws_sns_topic_subscription":        "free (topic itself bills via aws_sns_topic)",
	"aws_secretsmanager_secret_version": "free (secret bills via aws_secretsmanager_secret)",

	// Azure structural / free-by-design resources.
	"azurerm_recovery_services_vault":                  "free (protection bills on the protected resources)",
	"azurerm_data_lake_storage_gen2_filesystem":        "free (storage bills via the parent azurerm_storage_account)",
	"azurerm_network_security_group":                   "free",
	"azurerm_subnet":                                   "free",
	"azurerm_virtual_network":                          "free",
	"azurerm_route_table":                              "free",
	"azurerm_route":                                    "free",
	"azurerm_resource_group":                           "free",
	"azurerm_network_interface":                        "free",
	"azurerm_user_assigned_identity":                   "free",
	"azurerm_role_assignment":                          "free",
	"azurerm_key_vault_secret":                         "free (vault itself bills via azurerm_key_vault)",
	"azurerm_key_vault_access_policy":                  "free",
	"azurerm_storage_container":                        "free (storage bills via azurerm_storage_account)",
	"azurerm_storage_blob":                             "free at the resource level (data bills via the account)",
	"azurerm_mssql_server":                             "free (databases bill via azurerm_mssql_database)",
	"azurerm_postgresql_flexible_server_configuration": "free (server bills via azurerm_postgresql_flexible_server)",
	"azurerm_log_analytics_solution":                   "free (workspace bills via azurerm_log_analytics_workspace)",

	// GCP structural / free-by-design resources.
	"google_compute_network":                 "free",
	"google_compute_subnetwork":              "free",
	"google_compute_firewall":                "free",
	"google_compute_route":                   "free",
	"google_service_account":                 "free",
	"google_project_iam_member":              "free",
	"google_project_iam_binding":             "free",
	"google_project_iam_policy":              "free",
	"google_compute_target_pool":             "free (billed via underlying compute)",
	"google_compute_url_map":                 "free",
	"google_compute_backend_service":         "free (billed via attached LB / Cloud CDN)",
	"google_compute_health_check":            "free",
	"google_compute_router":                  "free (NAT gateway bills separately)",
	"google_compute_router_peer":             "free",
	"google_compute_global_address":          "free until detached (handled via google_compute_address)",
	"google_compute_forwarding_rule":         "free (billed via attached LB)",
	"google_compute_global_forwarding_rule":  "free",
	"google_kms_key_ring":                    "free (key bills via google_kms_crypto_key)",
	"google_kms_crypto_key_iam_member":       "free",
	"google_compute_instance_from_template":  "free (instance billing flows through google_compute_instance)",
	"google_certificate_manager_certificate": "free (Google-managed public TLS certificates carry no per-cert charge)",
}

// IsLegitimatelyFree reports whether the given kind appears in the
// LegitimatelyFreeKinds set.
func IsLegitimatelyFree(kind string) bool {
	_, ok := LegitimatelyFreeKinds[kind]
	return ok
}

// IsFreeShell reports whether the given kind's definition has only
// dimensions that quantify to 0 (i.e. a hand-authored TOML that
// declares the resource free-by-design). Used by the verifier to
// bucket bulk-added structural kinds (IAM, VPC plumbing) as [FREE]
// without growing the curated allowlist for each one.
func (r *Registry) IsFreeShell(kind string) bool {
	def := r.byKind[kind]
	if def == nil || len(def.Dimensions) == 0 {
		return false
	}
	for _, dim := range def.Dimensions {
		if dim.Quantity != "0" {
			return false
		}
	}
	return true
}

// HasStaticRate reports whether any dimension of the given Kind uses
// an inline literal rate instead of a `price()` lookup. Used by the
// verifier to bucket inline-rate resources as [STATIC].
func (r *Registry) HasStaticRate(kind string) bool {
	def := r.byKind[kind]
	if def == nil {
		return false
	}
	for _, dim := range def.Dimensions {
		// A literal zero is not a price: it marks a line that is billed
		// elsewhere (throughput shared with a parent database, a database
		// inside an elastic pool) and has no rate that could go stale.
		if isNumericLiteral(dim.Rate) && !isZeroLiteral(dim.Rate) {
			return true
		}
	}
	return false
}

// isZeroLiteral reports whether a numeric literal rate is zero ("0",
// "0.0", " 0 ").
func isZeroLiteral(expr string) bool {
	for _, c := range expr {
		switch c {
		case '0', '.', '+', '-', ' ', '\t':
		default:
			return false
		}
	}
	return true
}

// isNumericLiteral reports whether `expr` is a bare numeric literal
// (`"5.00"`, `"0.005"`) rather than an expression involving `price()`
// or arithmetic.
func isNumericLiteral(expr string) bool {
	// Cheap pre-check: any `price(` rules it out.
	if containsCall(expr, "price") {
		return false
	}
	// Try parsing as decimal. The cheap path: every character is digit,
	// dot, sign, or whitespace.
	for _, c := range expr {
		switch {
		case c == ' ' || c == '\t':
		case c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '+':
		default:
			return false
		}
	}
	return expr != ""
}

// containsCall reports whether `expr` calls `fn(` somewhere (cheap
// substring match, good enough for the literal/expression discriminator).
func containsCall(expr, fn string) bool {
	target := fn + "("
	for i := 0; i+len(target) <= len(expr); i++ {
		if expr[i:i+len(target)] == target {
			return true
		}
	}
	return false
}
