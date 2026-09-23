# Supported resources

Auto-generated from `resources/<provider>/*.toml`. Run
`go run ./cmd/gen_catalog_doc > docs/catalog.md` to refresh.

Status meaning:

- **LIVE** — priced against `pricing.c3x.dev`; tracks vendor changes.
- **STATIC** — inline rate (upstream catalog doesn't expose the meter).
  See `docs/upstream-gaps.md` for the per-resource explanation.
- **FREE** — no per-resource charge (parent-billed, structural, IAM).

## AWS (625 resources)

| Kind | Status | Display name |
|---|---|---|
| `aws_acm_certificate` | FREE | AWS ACM Certificate |
| `aws_acm_certificate_validation` | FREE | aws_acm_certificate_validation |
| `aws_acmpca_certificate` | FREE | aws_acmpca_certificate |
| `aws_acmpca_certificate_authority` | LIVE | AWS Private Certificate Authority |
| `aws_acmpca_certificate_authority_certificate` | FREE | aws_acmpca_certificate_authority_certificate |
| `aws_acmpca_permission` | FREE | aws_acmpca_permission |
| `aws_acmpca_policy` | FREE | aws_acmpca_policy |
| `aws_alb_listener` | FREE | aws_alb_listener |
| `aws_alb_listener_certificate` | FREE | aws_alb_listener_certificate |
| `aws_alb_listener_rule` | FREE | aws_alb_listener_rule |
| `aws_alb_target_group` | FREE | aws_alb_target_group |
| `aws_alb_target_group_attachment` | FREE | aws_alb_target_group_attachment |
| `aws_ami_launch_permission` | FREE | aws_ami_launch_permission |
| `aws_amplify_app` | LIVE | AWS Amplify Hosting App |
| `aws_amplify_backend_environment` | FREE | aws_amplify_backend_environment |
| `aws_amplify_branch` | FREE | aws_amplify_branch |
| `aws_amplify_domain_association` | FREE | aws_amplify_domain_association |
| `aws_amplify_webhook` | FREE | aws_amplify_webhook |
| `aws_api_gateway_api_key` | FREE | aws_api_gateway_api_key |
| `aws_api_gateway_authorizer` | FREE | aws_api_gateway_authorizer |
| `aws_api_gateway_base_path_mapping` | FREE | aws_api_gateway_base_path_mapping |
| `aws_api_gateway_client_certificate` | FREE | aws_api_gateway_client_certificate |
| `aws_api_gateway_deployment` | FREE | aws_api_gateway_deployment |
| `aws_api_gateway_documentation_part` | FREE | aws_api_gateway_documentation_part |
| `aws_api_gateway_documentation_version` | FREE | aws_api_gateway_documentation_version |
| `aws_api_gateway_domain_name` | FREE | aws_api_gateway_domain_name |
| `aws_api_gateway_gateway_response` | FREE | aws_api_gateway_gateway_response |
| `aws_api_gateway_integration` | FREE | aws_api_gateway_integration |
| `aws_api_gateway_integration_response` | FREE | aws_api_gateway_integration_response |
| `aws_api_gateway_method_response` | FREE | aws_api_gateway_method_response |
| `aws_api_gateway_method_settings` | FREE | aws_api_gateway_method_settings |
| `aws_api_gateway_model` | FREE | aws_api_gateway_model |
| `aws_api_gateway_request_validator` | FREE | aws_api_gateway_request_validator |
| `aws_api_gateway_rest_api` | LIVE | AWS API Gateway (REST) |
| `aws_api_gateway_rest_api_policy` | FREE | aws_api_gateway_rest_api_policy |
| `aws_api_gateway_usage_plan` | FREE | aws_api_gateway_usage_plan |
| `aws_api_gateway_usage_plan_key` | FREE | aws_api_gateway_usage_plan_key |
| `aws_api_gateway_vpc_link` | FREE | aws_api_gateway_vpc_link |
| `aws_apigateway_account` | FREE | AWS Apigateway Account |
| `aws_apigateway_method` | FREE | AWS Apigateway Method |
| `aws_apigateway_resource` | FREE | AWS Apigateway Resource |
| `aws_apigatewayv2_api` | LIVE | AWS API Gateway (HTTP/WebSocket) |
| `aws_apigatewayv2_api_mapping` | FREE | AWS Apigatewayv2 Api Mapping |
| `aws_apigatewayv2_authorizer` | FREE | AWS Apigatewayv2 Authorizer |
| `aws_apigatewayv2_deployment` | FREE | AWS Apigatewayv2 Deployment |
| `aws_apigatewayv2_domain_name` | FREE | aws_apigatewayv2_domain_name |
| `aws_apigatewayv2_integration` | FREE | AWS Apigatewayv2 Integration |
| `aws_apigatewayv2_integration_response` | FREE | aws_apigatewayv2_integration_response |
| `aws_apigatewayv2_model` | FREE | aws_apigatewayv2_model |
| `aws_apigatewayv2_route` | FREE | AWS Apigatewayv2 Route |
| `aws_apigatewayv2_route_response` | FREE | aws_apigatewayv2_route_response |
| `aws_apigatewayv2_stage` | FREE | AWS Apigatewayv2 Stage |
| `aws_apigatewayv2_vpc_link` | FREE | aws_apigatewayv2_vpc_link |
| `aws_appautoscaling_policy` | FREE | AWS Appautoscaling Policy |
| `aws_appautoscaling_target` | FREE | AWS Appautoscaling Target |
| `aws_appconfig_application` | FREE | aws_appconfig_application |
| `aws_appconfig_configuration_profile` | FREE | aws_appconfig_configuration_profile |
| `aws_appconfig_deployment` | FREE | aws_appconfig_deployment |
| `aws_appconfig_deployment_strategy` | FREE | aws_appconfig_deployment_strategy |
| `aws_appconfig_environment` | FREE | aws_appconfig_environment |
| `aws_appconfig_extension` | FREE | aws_appconfig_extension |
| `aws_appconfig_extension_association` | FREE | aws_appconfig_extension_association |
| `aws_appconfig_hosted_configuration_version` | FREE | aws_appconfig_hosted_configuration_version |
| `aws_appmesh_gateway_route` | FREE | aws_appmesh_gateway_route |
| `aws_appmesh_mesh` | FREE | aws_appmesh_mesh |
| `aws_appmesh_route` | FREE | aws_appmesh_route |
| `aws_appmesh_virtual_gateway` | FREE | aws_appmesh_virtual_gateway |
| `aws_appmesh_virtual_node` | FREE | aws_appmesh_virtual_node |
| `aws_appmesh_virtual_router` | FREE | aws_appmesh_virtual_router |
| `aws_appmesh_virtual_service` | FREE | aws_appmesh_virtual_service |
| `aws_apprunner_service` | LIVE | AWS App Runner Service |
| `aws_appstream_fleet` | LIVE | AWS AppStream 2.0 Fleet |
| `aws_appsync_graphql_api` | LIVE | AWS AppSync GraphQL API |
| `aws_athena_data_catalog` | FREE | aws_athena_data_catalog |
| `aws_athena_database` | FREE | aws_athena_database |
| `aws_athena_named_query` | FREE | aws_athena_named_query |
| `aws_athena_prepared_statement` | FREE | aws_athena_prepared_statement |
| `aws_athena_workgroup` | LIVE | AWS Athena Workgroup |
| `aws_autoscaling_attachment` | FREE | aws_autoscaling_attachment |
| `aws_autoscaling_group` | FREE | AWS Autoscaling Group |
| `aws_autoscaling_lifecycle_hook` | FREE | aws_autoscaling_lifecycle_hook |
| `aws_autoscaling_notification` | FREE | aws_autoscaling_notification |
| `aws_autoscaling_policy` | FREE | aws_autoscaling_policy |
| `aws_autoscaling_schedule` | FREE | aws_autoscaling_schedule |
| `aws_autoscaling_traffic_source_attachment` | FREE | aws_autoscaling_traffic_source_attachment |
| `aws_backup_framework` | FREE | aws_backup_framework |
| `aws_backup_global_settings` | FREE | aws_backup_global_settings |
| `aws_backup_plan` | FREE | aws_backup_plan |
| `aws_backup_region_settings` | FREE | aws_backup_region_settings |
| `aws_backup_report_plan` | FREE | aws_backup_report_plan |
| `aws_backup_selection` | FREE | aws_backup_selection |
| `aws_backup_vault` | LIVE | AWS Backup Vault |
| `aws_backup_vault_lock_configuration` | FREE | aws_backup_vault_lock_configuration |
| `aws_backup_vault_notifications` | FREE | aws_backup_vault_notifications |
| `aws_backup_vault_policy` | FREE | aws_backup_vault_policy |
| `aws_batch_compute_environment` | FREE | aws_batch_compute_environment |
| `aws_batch_job_definition` | FREE | aws_batch_job_definition |
| `aws_batch_job_queue` | FREE | aws_batch_job_queue |
| `aws_batch_scheduling_policy` | FREE | aws_batch_scheduling_policy |
| `aws_chatbot_slack_channel_configuration` | FREE | AWS Chatbot Slack Channel Configuration |
| `aws_cloudformation_export` | FREE | aws_cloudformation_export |
| `aws_cloudformation_stack_set_instance` | FREE | aws_cloudformation_stack_set_instance |
| `aws_cloudformation_type` | FREE | aws_cloudformation_type |
| `aws_cloudfront_cache_policy` | FREE | aws_cloudfront_cache_policy |
| `aws_cloudfront_continuous_deployment_policy` | FREE | aws_cloudfront_continuous_deployment_policy |
| `aws_cloudfront_distribution` | LIVE | AWS CloudFront Distribution |
| `aws_cloudfront_field_level_encryption_config` | FREE | aws_cloudfront_field_level_encryption_config |
| `aws_cloudfront_field_level_encryption_profile` | FREE | aws_cloudfront_field_level_encryption_profile |
| `aws_cloudfront_key_group` | FREE | aws_cloudfront_key_group |
| `aws_cloudfront_key_value_store` | FREE | aws_cloudfront_key_value_store |
| `aws_cloudfront_monitoring_subscription` | FREE | aws_cloudfront_monitoring_subscription |
| `aws_cloudfront_origin_access_control` | FREE | aws_cloudfront_origin_access_control |
| `aws_cloudfront_origin_access_identity` | FREE | aws_cloudfront_origin_access_identity |
| `aws_cloudfront_origin_request_policy` | FREE | aws_cloudfront_origin_request_policy |
| `aws_cloudfront_public_key` | FREE | aws_cloudfront_public_key |
| `aws_cloudfront_realtime_log_config` | FREE | aws_cloudfront_realtime_log_config |
| `aws_cloudfront_response_headers_policy` | FREE | aws_cloudfront_response_headers_policy |
| `aws_cloudfront_vpc_origin` | FREE | aws_cloudfront_vpc_origin |
| `aws_cloudhsm_v2_hsm` | LIVE | AWS CloudHSM v2 HSM |
| `aws_cloudtrail` | LIVE | AWS CloudTrail |
| `aws_cloudwatch_dashboard` | STATIC | AWS CloudWatch Dashboard |
| `aws_cloudwatch_event_api_destination` | FREE | aws_cloudwatch_event_api_destination |
| `aws_cloudwatch_event_archive` | FREE | aws_cloudwatch_event_archive |
| `aws_cloudwatch_event_bus_policy` | FREE | aws_cloudwatch_event_bus_policy |
| `aws_cloudwatch_event_connection` | FREE | aws_cloudwatch_event_connection |
| `aws_cloudwatch_event_endpoint` | FREE | aws_cloudwatch_event_endpoint |
| `aws_cloudwatch_event_permission` | FREE | aws_cloudwatch_event_permission |
| `aws_cloudwatch_event_rule` | STATIC | AWS CloudWatch Events Rule (EventBridge) |
| `aws_cloudwatch_event_target` | FREE | aws_cloudwatch_event_target |
| `aws_cloudwatch_log_account_policy` | FREE | aws_cloudwatch_log_account_policy |
| `aws_cloudwatch_log_data_protection_policy` | FREE | aws_cloudwatch_log_data_protection_policy |
| `aws_cloudwatch_log_destination_policy` | FREE | aws_cloudwatch_log_destination_policy |
| `aws_cloudwatch_log_group` | LIVE | AWS CloudWatch Log Group |
| `aws_cloudwatch_log_metric_filter` | FREE | AWS Cloudwatch Log Metric Filter |
| `aws_cloudwatch_log_resource_policy` | FREE | aws_cloudwatch_log_resource_policy |
| `aws_cloudwatch_log_stream` | FREE | AWS Cloudwatch Log Stream |
| `aws_cloudwatch_log_subscription_filter` | FREE | AWS Cloudwatch Log Subscription Filter |
| `aws_cloudwatch_metric_alarm` | LIVE | AWS CloudWatch Alarm |
| `aws_cloudwatch_query_definition` | FREE | aws_cloudwatch_query_definition |
| `aws_codeartifact_domain` | STATIC | AWS CodeArtifact Domain |
| `aws_codeartifact_domain_permissions_policy` | FREE | aws_codeartifact_domain_permissions_policy |
| `aws_codeartifact_repository_permissions_policy` | FREE | aws_codeartifact_repository_permissions_policy |
| `aws_codebuild_project` | LIVE | AWS CodeBuild Project |
| `aws_codebuild_report_group` | FREE | aws_codebuild_report_group |
| `aws_codebuild_source_credential` | FREE | aws_codebuild_source_credential |
| `aws_codebuild_webhook` | FREE | aws_codebuild_webhook |
| `aws_codecommit_approval_rule_template` | FREE | aws_codecommit_approval_rule_template |
| `aws_codecommit_approval_rule_template_association` | FREE | aws_codecommit_approval_rule_template_association |
| `aws_codecommit_repository` | FREE | AWS Codecommit Repository |
| `aws_codecommit_trigger` | FREE | aws_codecommit_trigger |
| `aws_codedeploy_app` | FREE | aws_codedeploy_app |
| `aws_codedeploy_application` | FREE | AWS Codedeploy Application |
| `aws_codedeploy_deployment_config` | FREE | aws_codedeploy_deployment_config |
| `aws_codedeploy_deployment_group` | FREE | AWS Codedeploy Deployment Group |
| `aws_codepipeline` | STATIC | AWS CodePipeline |
| `aws_codepipeline_custom_action_type` | FREE | aws_codepipeline_custom_action_type |
| `aws_codepipeline_webhook` | FREE | aws_codepipeline_webhook |
| `aws_codestar_connection` | FREE | AWS Codestar Connection |
| `aws_codestarconnections_connection` | FREE | aws_codestarconnections_connection |
| `aws_codestarconnections_host` | FREE | aws_codestarconnections_host |
| `aws_codestarnotifications_notification_rule` | FREE | aws_codestarnotifications_notification_rule |
| `aws_cognito_identity_pool` | FREE | aws_cognito_identity_pool |
| `aws_cognito_identity_pool_provider_principal_tag` | FREE | aws_cognito_identity_pool_provider_principal_tag |
| `aws_cognito_identity_pool_roles_attachment` | FREE | aws_cognito_identity_pool_roles_attachment |
| `aws_cognito_identity_provider` | FREE | aws_cognito_identity_provider |
| `aws_cognito_managed_user_pool_client` | FREE | aws_cognito_managed_user_pool_client |
| `aws_cognito_resource_server` | FREE | aws_cognito_resource_server |
| `aws_cognito_risk_configuration` | FREE | aws_cognito_risk_configuration |
| `aws_cognito_user` | FREE | aws_cognito_user |
| `aws_cognito_user_group` | FREE | aws_cognito_user_group |
| `aws_cognito_user_in_group` | FREE | aws_cognito_user_in_group |
| `aws_cognito_user_pool` | LIVE | AWS Cognito User Pool |
| `aws_cognito_user_pool_client` | FREE | aws_cognito_user_pool_client |
| `aws_cognito_user_pool_domain` | FREE | aws_cognito_user_pool_domain |
| `aws_cognito_user_pool_ui_customization` | FREE | aws_cognito_user_pool_ui_customization |
| `aws_config_config_rule` | LIVE | AWS Config Rule |
| `aws_config_configuration_recorder` | LIVE | AWS Config Recorder |
| `aws_config_delivery_channel` | FREE | AWS Config Delivery Channel |
| `aws_customer_gateway` | FREE | AWS Customer Gateway |
| `aws_datasync_agent` | FREE | aws_datasync_agent |
| `aws_datasync_location_efs` | FREE | aws_datasync_location_efs |
| `aws_datasync_location_fsx_lustre_file_system` | FREE | aws_datasync_location_fsx_lustre_file_system |
| `aws_datasync_location_nfs` | FREE | aws_datasync_location_nfs |
| `aws_datasync_location_s3` | FREE | aws_datasync_location_s3 |
| `aws_datasync_location_smb` | FREE | aws_datasync_location_smb |
| `aws_datasync_task` | FREE | aws_datasync_task |
| `aws_dax_cluster` | LIVE | AWS DAX Cluster |
| `aws_db_instance` | LIVE | AWS RDS Instance |
| `aws_db_option_group` | FREE | AWS Db Option Group |
| `aws_db_parameter_group` | FREE | AWS Db Parameter Group |
| `aws_db_proxy` | LIVE | AWS RDS Proxy |
| `aws_db_snapshot` | LIVE | AWS RDS DB Snapshot |
| `aws_db_subnet_group` | FREE | AWS Db Subnet Group |
| `aws_default_network_acl` | FREE | aws_default_network_acl |
| `aws_default_route_table` | FREE | aws_default_route_table |
| `aws_default_security_group` | FREE | aws_default_security_group |
| `aws_default_subnet` | FREE | aws_default_subnet |
| `aws_default_vpc` | FREE | aws_default_vpc |
| `aws_default_vpc_dhcp_options` | FREE | aws_default_vpc_dhcp_options |
| `aws_directconnect_gateway` | FREE | AWS Directconnect Gateway |
| `aws_directconnect_lag` | FREE | AWS Directconnect Lag |
| `aws_directory_service_directory` | LIVE | AWS Directory Service |
| `aws_dlm_lifecycle_policy` | FREE | aws_dlm_lifecycle_policy |
| `aws_dms_replication_instance` | LIVE | AWS DMS Replication Instance |
| `aws_documentdb_cluster` | LIVE | AWS DocumentDB Cluster |
| `aws_dx_bgp_peer` | FREE | aws_dx_bgp_peer |
| `aws_dx_connection` | LIVE | AWS Direct Connect Connection |
| `aws_dx_connection_association` | FREE | aws_dx_connection_association |
| `aws_dx_connection_confirmation` | FREE | aws_dx_connection_confirmation |
| `aws_dx_gateway_association` | FREE | aws_dx_gateway_association |
| `aws_dx_gateway_association_proposal` | FREE | aws_dx_gateway_association_proposal |
| `aws_dx_hosted_connection` | FREE | aws_dx_hosted_connection |
| `aws_dx_hosted_private_virtual_interface` | FREE | aws_dx_hosted_private_virtual_interface |
| `aws_dx_hosted_private_virtual_interface_accepter` | FREE | aws_dx_hosted_private_virtual_interface_accepter |
| `aws_dx_hosted_public_virtual_interface` | FREE | aws_dx_hosted_public_virtual_interface |
| `aws_dx_hosted_public_virtual_interface_accepter` | FREE | aws_dx_hosted_public_virtual_interface_accepter |
| `aws_dx_hosted_transit_virtual_interface` | FREE | aws_dx_hosted_transit_virtual_interface |
| `aws_dx_hosted_transit_virtual_interface_accepter` | FREE | aws_dx_hosted_transit_virtual_interface_accepter |
| `aws_dx_private_virtual_interface` | FREE | aws_dx_private_virtual_interface |
| `aws_dx_public_virtual_interface` | FREE | aws_dx_public_virtual_interface |
| `aws_dx_transit_virtual_interface` | FREE | aws_dx_transit_virtual_interface |
| `aws_dynamodb_resource_policy` | FREE | aws_dynamodb_resource_policy |
| `aws_dynamodb_table` | LIVE | AWS DynamoDB Table |
| `aws_dynamodb_table_item` | FREE | AWS Dynamodb Table Item |
| `aws_ebs_default_kms_key` | FREE | aws_ebs_default_kms_key |
| `aws_ebs_encryption_by_default` | FREE | aws_ebs_encryption_by_default |
| `aws_ebs_snapshot` | LIVE | AWS EBS Snapshot |
| `aws_ebs_snapshot_block_public_access` | FREE | aws_ebs_snapshot_block_public_access |
| `aws_ebs_volume` | LIVE | AWS EBS Volume |
| `aws_ec2_availability_zone_group` | FREE | aws_ec2_availability_zone_group |
| `aws_ec2_host` | LIVE | AWS EC2 Dedicated Host |
| `aws_ec2_image_block_public_access` | FREE | aws_ec2_image_block_public_access |
| `aws_ec2_instance_metadata_defaults` | FREE | aws_ec2_instance_metadata_defaults |
| `aws_ec2_instance_state` | FREE | aws_ec2_instance_state |
| `aws_ec2_managed_prefix_list` | FREE | aws_ec2_managed_prefix_list |
| `aws_ec2_managed_prefix_list_entry` | FREE | aws_ec2_managed_prefix_list_entry |
| `aws_ec2_serial_console_access` | FREE | aws_ec2_serial_console_access |
| `aws_ec2_subnet_cidr_reservation` | FREE | aws_ec2_subnet_cidr_reservation |
| `aws_ec2_tag` | FREE | aws_ec2_tag |
| `aws_ec2_transit_gateway_route` | FREE | aws_ec2_transit_gateway_route |
| `aws_ec2_transit_gateway_route_table` | FREE | aws_ec2_transit_gateway_route_table |
| `aws_ec2_transit_gateway_route_table_association` | FREE | aws_ec2_transit_gateway_route_table_association |
| `aws_ec2_transit_gateway_route_table_propagation` | FREE | aws_ec2_transit_gateway_route_table_propagation |
| `aws_ec2_transit_gateway_vpc_attachment` | STATIC | AWS Transit Gateway VPC Attachment |
| `aws_ecr_lifecycle_policy` | FREE | aws_ecr_lifecycle_policy |
| `aws_ecr_pull_through_cache_rule` | FREE | aws_ecr_pull_through_cache_rule |
| `aws_ecr_registry_policy` | FREE | aws_ecr_registry_policy |
| `aws_ecr_registry_scanning_configuration` | FREE | aws_ecr_registry_scanning_configuration |
| `aws_ecr_replication_configuration` | FREE | aws_ecr_replication_configuration |
| `aws_ecr_repository` | LIVE | AWS ECR Repository |
| `aws_ecr_repository_policy` | FREE | aws_ecr_repository_policy |
| `aws_ecrpublic_repository` | FREE | aws_ecrpublic_repository |
| `aws_ecrpublic_repository_policy` | FREE | aws_ecrpublic_repository_policy |
| `aws_ecs_account_setting_default` | FREE | aws_ecs_account_setting_default |
| `aws_ecs_capacity_provider` | FREE | aws_ecs_capacity_provider |
| `aws_ecs_cluster` | FREE | AWS Ecs Cluster |
| `aws_ecs_cluster_capacity_providers` | FREE | aws_ecs_cluster_capacity_providers |
| `aws_ecs_service` | LIVE | AWS ECS Service |
| `aws_ecs_tag` | FREE | aws_ecs_tag |
| `aws_ecs_task_definition` | FREE | AWS Ecs Task Definition |
| `aws_ecs_task_set` | FREE | aws_ecs_task_set |
| `aws_efs_access_point` | FREE | Efs Access Point |
| `aws_efs_file_system` | LIVE | AWS EFS File System |
| `aws_efs_file_system_policy` | FREE | aws_efs_file_system_policy |
| `aws_efs_mount_target` | FREE | Efs Mount Target |
| `aws_egress_only_internet_gateway` | FREE | aws_egress_only_internet_gateway |
| `aws_eip` | STATIC | AWS Elastic IP Address |
| `aws_eip_association` | FREE | AWS Eip Association |
| `aws_eks_access_entry` | FREE | aws_eks_access_entry |
| `aws_eks_access_policy_association` | FREE | aws_eks_access_policy_association |
| `aws_eks_addon` | FREE | aws_eks_addon |
| `aws_eks_cluster` | LIVE | AWS EKS Cluster |
| `aws_eks_fargate_profile` | FREE | AWS Eks Fargate Profile |
| `aws_eks_identity_provider_config` | FREE | aws_eks_identity_provider_config |
| `aws_eks_node_group` | LIVE | AWS EKS Node Group |
| `aws_eks_pod_identity_association` | FREE | aws_eks_pod_identity_association |
| `aws_elasticache_cluster` | LIVE | AWS ElastiCache Cluster |
| `aws_elasticache_parameter_group` | FREE | AWS Elasticache Parameter Group |
| `aws_elasticache_replication_group` | LIVE | AWS ElastiCache Replication Group |
| `aws_elasticache_serverless_cache` | LIVE | AWS ElastiCache Serverless Cache |
| `aws_elasticache_subnet_group` | FREE | AWS Elasticache Subnet Group |
| `aws_elasticsearch_domain` | LIVE | AWS Elasticsearch Domain (legacy kind) |
| `aws_elasticsearch_domain_policy` | FREE | aws_elasticsearch_domain_policy |
| `aws_emr_cluster` | LIVE | AWS EMR Cluster |
| `aws_fms_policy` | STATIC | AWS Firewall Manager Policy |
| `aws_fsx_lustre_file_system` | LIVE | AWS FSx for Lustre |
| `aws_fsx_ontap_file_system` | LIVE | AWS FSx for NetApp ONTAP |
| `aws_fsx_openzfs_file_system` | LIVE | AWS FSx for OpenZFS |
| `aws_fsx_windows_file_system` | LIVE | AWS FSx for Windows |
| `aws_glacier_vault` | LIVE | AWS S3 Glacier Vault |
| `aws_glacier_vault_lock` | FREE | aws_glacier_vault_lock |
| `aws_globalaccelerator_accelerator` | STATIC | AWS Global Accelerator |
| `aws_glue_catalog_database` | FREE | AWS Glue Data Catalog Database |
| `aws_glue_catalog_table` | FREE | aws_glue_catalog_table |
| `aws_glue_classifier` | FREE | aws_glue_classifier |
| `aws_glue_connection` | FREE | aws_glue_connection |
| `aws_glue_crawler` | LIVE | AWS Glue Crawler |
| `aws_glue_data_catalog_encryption_settings` | FREE | aws_glue_data_catalog_encryption_settings |
| `aws_glue_job` | LIVE | AWS Glue Job |
| `aws_glue_partition` | FREE | aws_glue_partition |
| `aws_glue_partition_index` | FREE | aws_glue_partition_index |
| `aws_glue_registry` | FREE | aws_glue_registry |
| `aws_glue_resource_policy` | FREE | aws_glue_resource_policy |
| `aws_glue_schema` | FREE | aws_glue_schema |
| `aws_glue_security_configuration` | FREE | aws_glue_security_configuration |
| `aws_glue_trigger` | FREE | aws_glue_trigger |
| `aws_glue_user_defined_function` | FREE | aws_glue_user_defined_function |
| `aws_glue_workflow` | FREE | aws_glue_workflow |
| `aws_grafana_workspace` | LIVE | AWS Managed Grafana Workspace |
| `aws_guardduty_detector` | STATIC | AWS GuardDuty Detector |
| `aws_guardduty_filter` | FREE | aws_guardduty_filter |
| `aws_guardduty_invite_accepter` | FREE | aws_guardduty_invite_accepter |
| `aws_guardduty_ipset` | FREE | aws_guardduty_ipset |
| `aws_guardduty_member` | FREE | aws_guardduty_member |
| `aws_guardduty_organization_admin_account` | FREE | aws_guardduty_organization_admin_account |
| `aws_guardduty_organization_configuration` | FREE | aws_guardduty_organization_configuration |
| `aws_guardduty_publishing_destination` | FREE | aws_guardduty_publishing_destination |
| `aws_guardduty_threatintelset` | FREE | aws_guardduty_threatintelset |
| `aws_iam_access_key` | FREE | AWS Iam Access Key |
| `aws_iam_account_alias` | FREE | aws_iam_account_alias |
| `aws_iam_account_password_policy` | FREE | aws_iam_account_password_policy |
| `aws_iam_group` | FREE | AWS Iam Group |
| `aws_iam_group_membership` | FREE | aws_iam_group_membership |
| `aws_iam_group_policy` | FREE | aws_iam_group_policy |
| `aws_iam_group_policy_attachment` | FREE | aws_iam_group_policy_attachment |
| `aws_iam_instance_profile` | FREE | AWS Iam Instance Profile |
| `aws_iam_openid_connect_provider` | FREE | aws_iam_openid_connect_provider |
| `aws_iam_policy` | FREE | AWS Iam Policy |
| `aws_iam_policy_attachment` | FREE | aws_iam_policy_attachment |
| `aws_iam_role` | FREE | Iam Role |
| `aws_iam_role_policy` | FREE | AWS Iam Role Policy |
| `aws_iam_role_policy_attachment` | FREE | AWS Iam Role Policy Attachment |
| `aws_iam_saml_provider` | FREE | aws_iam_saml_provider |
| `aws_iam_security_token_service_preferences` | FREE | aws_iam_security_token_service_preferences |
| `aws_iam_server_certificate` | FREE | aws_iam_server_certificate |
| `aws_iam_service_linked_role` | FREE | aws_iam_service_linked_role |
| `aws_iam_service_specific_credential` | FREE | aws_iam_service_specific_credential |
| `aws_iam_signing_certificate` | FREE | aws_iam_signing_certificate |
| `aws_iam_user` | FREE | AWS Iam User |
| `aws_iam_user_group_membership` | FREE | aws_iam_user_group_membership |
| `aws_iam_user_login_profile` | FREE | aws_iam_user_login_profile |
| `aws_iam_user_policy` | FREE | aws_iam_user_policy |
| `aws_iam_user_policy_attachment` | FREE | aws_iam_user_policy_attachment |
| `aws_iam_user_ssh_key` | FREE | aws_iam_user_ssh_key |
| `aws_iam_virtual_mfa_device` | FREE | aws_iam_virtual_mfa_device |
| `aws_inspector_assessment_target` | FREE | aws_inspector_assessment_target |
| `aws_inspector_assessment_template` | FREE | aws_inspector_assessment_template |
| `aws_inspector_resource_group` | FREE | aws_inspector_resource_group |
| `aws_instance` | LIVE | AWS EC2 Instance |
| `aws_internet_gateway` | FREE | Internet Gateway |
| `aws_iot_certificate` | FREE | aws_iot_certificate |
| `aws_iot_event_configurations` | FREE | aws_iot_event_configurations |
| `aws_iot_logging_options` | FREE | aws_iot_logging_options |
| `aws_iot_policy` | FREE | aws_iot_policy |
| `aws_iot_policy_attachment` | FREE | aws_iot_policy_attachment |
| `aws_iot_role_alias` | FREE | aws_iot_role_alias |
| `aws_iot_thing` | FREE | aws_iot_thing |
| `aws_iot_thing_group` | FREE | aws_iot_thing_group |
| `aws_iot_thing_group_membership` | FREE | aws_iot_thing_group_membership |
| `aws_iot_thing_principal_attachment` | FREE | aws_iot_thing_principal_attachment |
| `aws_iot_thing_type` | FREE | aws_iot_thing_type |
| `aws_iot_topic_rule` | FREE | aws_iot_topic_rule |
| `aws_kendra_index` | LIVE | AWS Kendra Index |
| `aws_key_pair` | FREE | aws_key_pair |
| `aws_kinesis_firehose_delivery_stream` | LIVE | AWS Kinesis Data Firehose |
| `aws_kinesis_resource_policy` | FREE | aws_kinesis_resource_policy |
| `aws_kinesis_stream` | LIVE | AWS Kinesis Data Stream |
| `aws_kinesis_stream_consumer` | FREE | aws_kinesis_stream_consumer |
| `aws_kinesisanalyticsv2_application` | LIVE | AWS Managed Flink Application |
| `aws_kms_alias` | FREE | AWS Kms Alias |
| `aws_kms_ciphertext` | FREE | aws_kms_ciphertext |
| `aws_kms_custom_key_store` | FREE | aws_kms_custom_key_store |
| `aws_kms_external_key` | FREE | aws_kms_external_key |
| `aws_kms_grant` | FREE | AWS Kms Grant |
| `aws_kms_key` | LIVE | AWS KMS Key |
| `aws_kms_key_policy` | FREE | aws_kms_key_policy |
| `aws_kms_replica_external_key` | FREE | aws_kms_replica_external_key |
| `aws_lambda_alias` | FREE | AWS Lambda Alias |
| `aws_lambda_code_signing_config` | FREE | aws_lambda_code_signing_config |
| `aws_lambda_event_source_mapping` | FREE | AWS Lambda Event Source Mapping |
| `aws_lambda_function` | LIVE | AWS Lambda Function |
| `aws_lambda_function_event_invoke_config` | FREE | aws_lambda_function_event_invoke_config |
| `aws_lambda_function_url` | FREE | AWS Lambda Function Url |
| `aws_lambda_layer_version` | FREE | AWS Lambda Layer Version |
| `aws_lambda_layer_version_permission` | FREE | aws_lambda_layer_version_permission |
| `aws_lambda_permission` | FREE | AWS Lambda Permission |
| `aws_lambda_runtime_management_config` | FREE | aws_lambda_runtime_management_config |
| `aws_launch_configuration` | FREE | AWS Launch Configuration |
| `aws_launch_template` | FREE | AWS Launch Template |
| `aws_launch_template_default_version` | FREE | aws_launch_template_default_version |
| `aws_lb` | LIVE | AWS Elastic Load Balancer |
| `aws_lb_listener` | FREE | AWS Lb Listener |
| `aws_lb_listener_certificate` | FREE | aws_lb_listener_certificate |
| `aws_lb_listener_rule` | FREE | aws_lb_listener_rule |
| `aws_lb_target_group` | FREE | AWS Load Balancer Target Group |
| `aws_lb_target_group_attachment` | FREE | aws_lb_target_group_attachment |
| `aws_lb_trust_store` | FREE | aws_lb_trust_store |
| `aws_lightsail_bucket` | LIVE | AWS Lightsail Bucket |
| `aws_lightsail_database` | LIVE | AWS Lightsail Database |
| `aws_lightsail_distribution` | LIVE | AWS Lightsail Distribution |
| `aws_lightsail_instance` | LIVE | AWS Lightsail Instance |
| `aws_macie2_account` | STATIC | AWS Macie Account |
| `aws_main_route_table_association` | FREE | AWS Main Route Table Association |
| `aws_memorydb_cluster` | LIVE | AWS MemoryDB Cluster |
| `aws_mq_broker` | LIVE | AWS MQ Broker |
| `aws_mq_configuration` | FREE | aws_mq_configuration |
| `aws_msk_cluster` | LIVE | AWS MSK Cluster (Kafka) |
| `aws_msk_cluster_policy` | FREE | aws_msk_cluster_policy |
| `aws_msk_configuration` | FREE | aws_msk_configuration |
| `aws_msk_scram_secret_association` | FREE | aws_msk_scram_secret_association |
| `aws_msk_serverless_cluster` | LIVE | AWS MSK Serverless Cluster |
| `aws_msk_vpc_connection` | FREE | aws_msk_vpc_connection |
| `aws_mwaa_environment` | LIVE | AWS MWAA Environment |
| `aws_nat_gateway` | STATIC | AWS NAT Gateway |
| `aws_neptune_cluster` | LIVE | AWS Neptune Cluster (graph) |
| `aws_neptune_cluster_instance` | LIVE | AWS Neptune Cluster Instance |
| `aws_network_acl` | FREE | aws_network_acl |
| `aws_network_acl_association` | FREE | aws_network_acl_association |
| `aws_network_acl_rule` | FREE | aws_network_acl_rule |
| `aws_network_interface` | FREE | AWS Network Interface |
| `aws_network_interface_attachment` | FREE | aws_network_interface_attachment |
| `aws_network_interface_sg_attachment` | FREE | aws_network_interface_sg_attachment |
| `aws_networkfirewall_firewall` | LIVE | AWS Network Firewall |
| `aws_opensearch_domain` | LIVE | AWS OpenSearch Service Domain |
| `aws_opensearch_domain_policy` | FREE | aws_opensearch_domain_policy |
| `aws_opensearchserverless_collection` | STATIC | AWS OpenSearch Serverless Collection |
| `aws_organizations_account` | FREE | AWS Organizations Account |
| `aws_organizations_delegated_administrator` | FREE | aws_organizations_delegated_administrator |
| `aws_organizations_organization` | FREE | AWS Organizations Organization |
| `aws_organizations_organizational_unit` | FREE | AWS Organizations Organizational Unit |
| `aws_organizations_policy` | FREE | AWS Organizations Policy |
| `aws_organizations_policy_attachment` | FREE | aws_organizations_policy_attachment |
| `aws_organizations_resource_policy` | FREE | aws_organizations_resource_policy |
| `aws_placement_group` | FREE | aws_placement_group |
| `aws_prometheus_workspace` | STATIC | AWS Managed Prometheus Workspace |
| `aws_prometheus_workspace_policy` | FREE | aws_prometheus_workspace_policy |
| `aws_qldb_ledger` | STATIC | AWS QLDB Ledger |
| `aws_ram_principal_association` | FREE | aws_ram_principal_association |
| `aws_ram_resource_association` | FREE | aws_ram_resource_association |
| `aws_ram_resource_share` | FREE | aws_ram_resource_share |
| `aws_ram_resource_share_accepter` | FREE | aws_ram_resource_share_accepter |
| `aws_ram_sharing_with_organization` | FREE | aws_ram_sharing_with_organization |
| `aws_rds_cluster` | LIVE | AWS Aurora Cluster |
| `aws_rds_cluster_instance` | LIVE | AWS Aurora Cluster Instance |
| `aws_redshift_cluster` | LIVE | AWS Redshift Cluster |
| `aws_redshift_parameter_group` | FREE | AWS Redshift Parameter Group |
| `aws_redshift_resource_policy` | FREE | aws_redshift_resource_policy |
| `aws_redshift_subnet_group` | FREE | AWS Redshift Subnet Group |
| `aws_redshiftserverless_workgroup` | STATIC | AWS Redshift Serverless Workgroup |
| `aws_resourceexplorer2_index` | FREE | aws_resourceexplorer2_index |
| `aws_resourceexplorer2_view` | FREE | aws_resourceexplorer2_view |
| `aws_resourcegroups_group` | FREE | aws_resourcegroups_group |
| `aws_route` | FREE | AWS Route |
| `aws_route53_cidr_collection` | FREE | aws_route53_cidr_collection |
| `aws_route53_cidr_location` | FREE | aws_route53_cidr_location |
| `aws_route53_delegation_set` | FREE | aws_route53_delegation_set |
| `aws_route53_health_check` | LIVE | AWS Route 53 Health Check |
| `aws_route53_hosted_zone_dnssec` | FREE | aws_route53_hosted_zone_dnssec |
| `aws_route53_key_signing_key` | FREE | aws_route53_key_signing_key |
| `aws_route53_query_log` | FREE | AWS Route53 Query Log |
| `aws_route53_record` | FREE | AWS Route53 Record |
| `aws_route53_resolver_config` | FREE | aws_route53_resolver_config |
| `aws_route53_resolver_dnssec_config` | FREE | aws_route53_resolver_dnssec_config |
| `aws_route53_resolver_endpoint` | STATIC | AWS Route 53 Resolver Endpoint |
| `aws_route53_resolver_firewall_config` | FREE | aws_route53_resolver_firewall_config |
| `aws_route53_resolver_rule_association` | FREE | aws_route53_resolver_rule_association |
| `aws_route53_traffic_policy` | FREE | aws_route53_traffic_policy |
| `aws_route53_vpc_association_authorization` | FREE | aws_route53_vpc_association_authorization |
| `aws_route53_zone` | LIVE | AWS Route53 Hosted Zone |
| `aws_route53_zone_association` | FREE | AWS Route53 Zone Association |
| `aws_route_table` | FREE | AWS Route Table |
| `aws_route_table_association` | FREE | aws_route_table_association |
| `aws_s3_access_point` | FREE | aws_s3_access_point |
| `aws_s3_account_public_access_block` | FREE | aws_s3_account_public_access_block |
| `aws_s3_bucket` | LIVE | AWS S3 Bucket |
| `aws_s3_bucket_accelerate_configuration` | FREE | aws_s3_bucket_accelerate_configuration |
| `aws_s3_bucket_acl` | FREE | AWS S3 Bucket Acl |
| `aws_s3_bucket_analytics_configuration` | FREE | aws_s3_bucket_analytics_configuration |
| `aws_s3_bucket_cors_configuration` | FREE | AWS S3 Bucket Cors Configuration |
| `aws_s3_bucket_intelligent_tiering_configuration` | FREE | aws_s3_bucket_intelligent_tiering_configuration |
| `aws_s3_bucket_inventory` | FREE | aws_s3_bucket_inventory |
| `aws_s3_bucket_lifecycle_configuration` | FREE | AWS S3 Bucket Lifecycle Configuration |
| `aws_s3_bucket_logging` | FREE | AWS S3 Bucket Logging |
| `aws_s3_bucket_metric` | FREE | aws_s3_bucket_metric |
| `aws_s3_bucket_notification` | FREE | AWS S3 Bucket Notification |
| `aws_s3_bucket_object_lock_configuration` | FREE | aws_s3_bucket_object_lock_configuration |
| `aws_s3_bucket_ownership_controls` | FREE | aws_s3_bucket_ownership_controls |
| `aws_s3_bucket_policy` | FREE | AWS S3 Bucket Policy |
| `aws_s3_bucket_public_access_block` | FREE | AWS S3 Bucket Public Access Block |
| `aws_s3_bucket_replication_configuration` | FREE | aws_s3_bucket_replication_configuration |
| `aws_s3_bucket_request_payment_configuration` | FREE | aws_s3_bucket_request_payment_configuration |
| `aws_s3_bucket_server_side_encryption_configuration` | FREE | AWS S3 Bucket Server Side Encryption Configuration |
| `aws_s3_bucket_versioning` | FREE | AWS S3 Bucket Versioning |
| `aws_s3_bucket_website_configuration` | FREE | aws_s3_bucket_website_configuration |
| `aws_s3_object` | FREE | AWS S3 Object |
| `aws_s3control_access_point_policy` | FREE | aws_s3control_access_point_policy |
| `aws_s3control_bucket_policy` | FREE | aws_s3control_bucket_policy |
| `aws_sagemaker_endpoint` | STATIC | AWS SageMaker Endpoint |
| `aws_sagemaker_notebook_instance` | LIVE | AWS SageMaker Notebook Instance |
| `aws_scheduler_schedule` | FREE | aws_scheduler_schedule |
| `aws_scheduler_schedule_group` | FREE | aws_scheduler_schedule_group |
| `aws_schemas_discoverer` | FREE | aws_schemas_discoverer |
| `aws_schemas_registry` | FREE | aws_schemas_registry |
| `aws_schemas_registry_policy` | FREE | aws_schemas_registry_policy |
| `aws_schemas_schema` | FREE | aws_schemas_schema |
| `aws_secretsmanager_secret` | LIVE | AWS Secrets Manager Secret |
| `aws_secretsmanager_secret_policy` | FREE | aws_secretsmanager_secret_policy |
| `aws_secretsmanager_secret_rotation` | FREE | aws_secretsmanager_secret_rotation |
| `aws_secretsmanager_secret_version` | FREE | AWS Secretsmanager Secret Version |
| `aws_security_group` | FREE | Security Group |
| `aws_security_group_rule` | FREE | aws_security_group_rule |
| `aws_securityhub_account` | STATIC | AWS Security Hub Account |
| `aws_service_discovery_http_namespace` | FREE | aws_service_discovery_http_namespace |
| `aws_service_discovery_instance` | FREE | aws_service_discovery_instance |
| `aws_service_discovery_private_dns_namespace` | FREE | aws_service_discovery_private_dns_namespace |
| `aws_service_discovery_public_dns_namespace` | FREE | aws_service_discovery_public_dns_namespace |
| `aws_service_discovery_service` | FREE | aws_service_discovery_service |
| `aws_servicecatalog_constraint` | FREE | aws_servicecatalog_constraint |
| `aws_servicecatalog_portfolio` | FREE | AWS Servicecatalog Portfolio |
| `aws_servicecatalog_portfolio_share` | FREE | aws_servicecatalog_portfolio_share |
| `aws_servicecatalog_principal_portfolio_association` | FREE | aws_servicecatalog_principal_portfolio_association |
| `aws_servicecatalog_product` | FREE | AWS Servicecatalog Product |
| `aws_servicecatalog_product_portfolio_association` | FREE | aws_servicecatalog_product_portfolio_association |
| `aws_servicecatalog_provisioning_artifact` | FREE | aws_servicecatalog_provisioning_artifact |
| `aws_servicecatalog_tag_option` | FREE | aws_servicecatalog_tag_option |
| `aws_ses_configuration_set` | FREE | aws_ses_configuration_set |
| `aws_ses_domain_dkim` | FREE | aws_ses_domain_dkim |
| `aws_ses_domain_identity` | FREE | aws_ses_domain_identity |
| `aws_ses_domain_identity_verification` | FREE | aws_ses_domain_identity_verification |
| `aws_ses_domain_mail_from` | FREE | aws_ses_domain_mail_from |
| `aws_ses_email_identity` | FREE | aws_ses_email_identity |
| `aws_ses_event_destination` | FREE | aws_ses_event_destination |
| `aws_ses_identity_policy` | FREE | aws_ses_identity_policy |
| `aws_ses_receipt_filter` | FREE | aws_ses_receipt_filter |
| `aws_ses_receipt_rule` | FREE | aws_ses_receipt_rule |
| `aws_ses_receipt_rule_set` | FREE | aws_ses_receipt_rule_set |
| `aws_ses_template` | FREE | aws_ses_template |
| `aws_sesv2_configuration_set` | FREE | aws_sesv2_configuration_set |
| `aws_sesv2_configuration_set_event_destination` | FREE | aws_sesv2_configuration_set_event_destination |
| `aws_sesv2_contact_list` | FREE | aws_sesv2_contact_list |
| `aws_sesv2_dedicated_ip_pool` | FREE | aws_sesv2_dedicated_ip_pool |
| `aws_sesv2_email_identity` | FREE | aws_sesv2_email_identity |
| `aws_sesv2_email_identity_dkim_signing_attributes` | FREE | aws_sesv2_email_identity_dkim_signing_attributes |
| `aws_sesv2_email_identity_feedback_attributes` | FREE | aws_sesv2_email_identity_feedback_attributes |
| `aws_sesv2_email_identity_mail_from_attributes` | FREE | aws_sesv2_email_identity_mail_from_attributes |
| `aws_sesv2_email_identity_policy` | FREE | aws_sesv2_email_identity_policy |
| `aws_sfn_activity` | FREE | aws_sfn_activity |
| `aws_sfn_alias` | FREE | aws_sfn_alias |
| `aws_sfn_state_machine` | LIVE | AWS Step Functions State Machine |
| `aws_shield_protection` | LIVE | AWS Shield Advanced Protection |
| `aws_snapshot_create_volume_permission` | FREE | aws_snapshot_create_volume_permission |
| `aws_sns_platform_application` | FREE | aws_sns_platform_application |
| `aws_sns_sms_preferences` | FREE | aws_sns_sms_preferences |
| `aws_sns_topic` | LIVE | AWS SNS Topic |
| `aws_sns_topic_policy` | FREE | aws_sns_topic_policy |
| `aws_sns_topic_subscription` | FREE | AWS Sns Topic Subscription |
| `aws_spot_datafeed_subscription` | FREE | aws_spot_datafeed_subscription |
| `aws_spot_instance_request` | LIVE | AWS Spot Instance Request |
| `aws_sqs_queue` | LIVE | AWS SQS Queue |
| `aws_sqs_queue_policy` | FREE | AWS Sqs Queue Policy |
| `aws_sqs_queue_redrive_allow_policy` | FREE | aws_sqs_queue_redrive_allow_policy |
| `aws_sqs_queue_redrive_policy` | FREE | aws_sqs_queue_redrive_policy |
| `aws_ssm_association` | FREE | AWS Ssm Association |
| `aws_ssm_default_patch_baseline` | FREE | aws_ssm_default_patch_baseline |
| `aws_ssm_document` | FREE | AWS Ssm Document |
| `aws_ssm_maintenance_window` | FREE | AWS Ssm Maintenance Window |
| `aws_ssm_maintenance_window_target` | FREE | aws_ssm_maintenance_window_target |
| `aws_ssm_maintenance_window_task` | FREE | aws_ssm_maintenance_window_task |
| `aws_ssm_parameter` | LIVE | AWS SSM Parameter Store Parameter |
| `aws_ssm_patch_baseline` | FREE | AWS Ssm Patch Baseline |
| `aws_ssm_patch_group` | FREE | aws_ssm_patch_group |
| `aws_ssm_resource_data_sync` | FREE | aws_ssm_resource_data_sync |
| `aws_ssm_service_setting` | FREE | aws_ssm_service_setting |
| `aws_storagegateway_gateway` | STATIC | AWS Storage Gateway |
| `aws_subnet` | FREE | AWS Subnet |
| `aws_timestreamwrite_table` | STATIC | AWS Timestream Table |
| `aws_transfer_access` | FREE | aws_transfer_access |
| `aws_transfer_server` | LIVE | AWS Transfer Family Server |
| `aws_transfer_ssh_key` | FREE | aws_transfer_ssh_key |
| `aws_transfer_tag` | FREE | aws_transfer_tag |
| `aws_transfer_user` | FREE | aws_transfer_user |
| `aws_transfer_workflow` | FREE | aws_transfer_workflow |
| `aws_transit_gateway` | LIVE | AWS Transit Gateway |
| `aws_volume_attachment` | FREE | aws_volume_attachment |
| `aws_vpc` | FREE | AWS Vpc |
| `aws_vpc_dhcp_options` | FREE | aws_vpc_dhcp_options |
| `aws_vpc_dhcp_options_association` | FREE | aws_vpc_dhcp_options_association |
| `aws_vpc_endpoint` | LIVE | AWS VPC Endpoint |
| `aws_vpc_endpoint_connection_accepter` | FREE | aws_vpc_endpoint_connection_accepter |
| `aws_vpc_endpoint_connection_notification` | FREE | aws_vpc_endpoint_connection_notification |
| `aws_vpc_endpoint_policy` | FREE | aws_vpc_endpoint_policy |
| `aws_vpc_endpoint_route_table_association` | FREE | aws_vpc_endpoint_route_table_association |
| `aws_vpc_endpoint_security_group_association` | FREE | aws_vpc_endpoint_security_group_association |
| `aws_vpc_endpoint_service` | FREE | aws_vpc_endpoint_service |
| `aws_vpc_endpoint_service_allowed_principal` | FREE | aws_vpc_endpoint_service_allowed_principal |
| `aws_vpc_endpoint_subnet_association` | FREE | aws_vpc_endpoint_subnet_association |
| `aws_vpc_ipv4_cidr_block_association` | FREE | aws_vpc_ipv4_cidr_block_association |
| `aws_vpc_ipv6_cidr_block_association` | FREE | aws_vpc_ipv6_cidr_block_association |
| `aws_vpc_peering_connection` | FREE | AWS Vpc Peering Connection |
| `aws_vpc_peering_connection_accepter` | FREE | aws_vpc_peering_connection_accepter |
| `aws_vpc_peering_connection_options` | FREE | aws_vpc_peering_connection_options |
| `aws_vpc_security_group_egress_rule` | FREE | aws_vpc_security_group_egress_rule |
| `aws_vpc_security_group_ingress_rule` | FREE | aws_vpc_security_group_ingress_rule |
| `aws_vpn_connection` | LIVE | AWS Site-to-Site VPN Connection |
| `aws_vpn_gateway` | FREE | AWS Vpn Gateway |
| `aws_waf_byte_match_set` | FREE | aws_waf_byte_match_set |
| `aws_waf_geo_match_set` | FREE | aws_waf_geo_match_set |
| `aws_waf_ipset` | FREE | aws_waf_ipset |
| `aws_waf_rate_based_rule` | FREE | aws_waf_rate_based_rule |
| `aws_waf_regex_match_set` | FREE | aws_waf_regex_match_set |
| `aws_waf_regex_pattern_set` | FREE | aws_waf_regex_pattern_set |
| `aws_waf_rule` | FREE | aws_waf_rule |
| `aws_waf_rule_group` | FREE | aws_waf_rule_group |
| `aws_waf_size_constraint_set` | FREE | aws_waf_size_constraint_set |
| `aws_waf_sql_injection_match_set` | FREE | aws_waf_sql_injection_match_set |
| `aws_waf_web_acl` | FREE | aws_waf_web_acl |
| `aws_waf_xss_match_set` | FREE | aws_waf_xss_match_set |
| `aws_wafv2_ip_set` | FREE | aws_wafv2_ip_set |
| `aws_wafv2_regex_pattern_set` | FREE | aws_wafv2_regex_pattern_set |
| `aws_wafv2_rule_group` | FREE | aws_wafv2_rule_group |
| `aws_wafv2_web_acl` | LIVE | AWS WAFv2 Web ACL |
| `aws_wafv2_web_acl_association` | FREE | aws_wafv2_web_acl_association |
| `aws_wafv2_web_acl_logging_configuration` | FREE | aws_wafv2_web_acl_logging_configuration |
| `aws_workspaces_workspace` | LIVE | AWS WorkSpaces Desktop |
| `aws_xray_group` | LIVE | AWS X-Ray Group |

**AWS totals:** 90 LIVE · 20 STATIC · 515 FREE

## AZURE (409 resources)

| Kind | Status | Display name |
|---|---|---|
| `azurerm_aadb2c_directory` | FREE | azurerm_aadb2c_directory |
| `azurerm_analysis_services_server` | LIVE | Azure Analysis Services Server |
| `azurerm_api_management` | LIVE | Azure API Management |
| `azurerm_api_management_api` | FREE | azurerm_api_management_api |
| `azurerm_api_management_api_diagnostic` | FREE | azurerm_api_management_api_diagnostic |
| `azurerm_api_management_api_operation` | FREE | azurerm_api_management_api_operation |
| `azurerm_api_management_api_operation_policy` | FREE | azurerm_api_management_api_operation_policy |
| `azurerm_api_management_api_policy` | FREE | azurerm_api_management_api_policy |
| `azurerm_api_management_api_schema` | FREE | azurerm_api_management_api_schema |
| `azurerm_api_management_api_version_set` | FREE | azurerm_api_management_api_version_set |
| `azurerm_api_management_authorization_server` | FREE | azurerm_api_management_authorization_server |
| `azurerm_api_management_backend` | FREE | azurerm_api_management_backend |
| `azurerm_api_management_certificate` | FREE | azurerm_api_management_certificate |
| `azurerm_api_management_custom_domain` | FREE | azurerm_api_management_custom_domain |
| `azurerm_api_management_diagnostic` | FREE | azurerm_api_management_diagnostic |
| `azurerm_api_management_group` | FREE | azurerm_api_management_group |
| `azurerm_api_management_group_user` | FREE | azurerm_api_management_group_user |
| `azurerm_api_management_identity_provider_aad` | FREE | azurerm_api_management_identity_provider_aad |
| `azurerm_api_management_logger` | FREE | azurerm_api_management_logger |
| `azurerm_api_management_named_value` | FREE | azurerm_api_management_named_value |
| `azurerm_api_management_notification_recipient_email` | FREE | azurerm_api_management_notification_recipient_email |
| `azurerm_api_management_policy` | FREE | azurerm_api_management_policy |
| `azurerm_api_management_product` | FREE | azurerm_api_management_product |
| `azurerm_api_management_product_api` | FREE | azurerm_api_management_product_api |
| `azurerm_api_management_product_group` | FREE | azurerm_api_management_product_group |
| `azurerm_api_management_product_policy` | FREE | azurerm_api_management_product_policy |
| `azurerm_api_management_subscription` | FREE | azurerm_api_management_subscription |
| `azurerm_api_management_user` | FREE | azurerm_api_management_user |
| `azurerm_app_configuration` | LIVE | Azure App Configuration |
| `azurerm_app_service_certificate_binding` | FREE | azurerm_app_service_certificate_binding |
| `azurerm_app_service_custom_hostname_binding` | FREE | Azure App Service Custom Hostname Binding |
| `azurerm_app_service_environment_v3` | FREE | azurerm_app_service_environment_v3 |
| `azurerm_app_service_slot_custom_hostname_binding` | FREE | azurerm_app_service_slot_custom_hostname_binding |
| `azurerm_app_service_source_control` | FREE | azurerm_app_service_source_control |
| `azurerm_app_service_source_control_slot` | FREE | azurerm_app_service_source_control_slot |
| `azurerm_app_service_source_control_token` | FREE | azurerm_app_service_source_control_token |
| `azurerm_application_gateway` | LIVE | Azure Application Gateway |
| `azurerm_application_insights` | STATIC | Azure Application Insights |
| `azurerm_application_insights_analytics_item` | FREE | azurerm_application_insights_analytics_item |
| `azurerm_application_insights_api_key` | FREE | azurerm_application_insights_api_key |
| `azurerm_application_insights_smart_detection_rule` | FREE | azurerm_application_insights_smart_detection_rule |
| `azurerm_application_insights_web_test` | FREE | azurerm_application_insights_web_test |
| `azurerm_application_insights_workbook` | FREE | azurerm_application_insights_workbook |
| `azurerm_application_insights_workbook_template` | FREE | azurerm_application_insights_workbook_template |
| `azurerm_application_security_group` | FREE | azurerm_application_security_group |
| `azurerm_automation_account` | STATIC | Azure Automation Account |
| `azurerm_automation_certificate` | FREE | azurerm_automation_certificate |
| `azurerm_automation_connection` | FREE | azurerm_automation_connection |
| `azurerm_automation_credential` | FREE | azurerm_automation_credential |
| `azurerm_automation_job_schedule` | FREE | azurerm_automation_job_schedule |
| `azurerm_automation_module` | FREE | azurerm_automation_module |
| `azurerm_automation_runbook` | FREE | azurerm_automation_runbook |
| `azurerm_automation_schedule` | FREE | azurerm_automation_schedule |
| `azurerm_automation_variable_bool` | FREE | azurerm_automation_variable_bool |
| `azurerm_automation_variable_datetime` | FREE | azurerm_automation_variable_datetime |
| `azurerm_automation_variable_int` | FREE | azurerm_automation_variable_int |
| `azurerm_automation_variable_string` | FREE | azurerm_automation_variable_string |
| `azurerm_backup_protected_vm` | STATIC | Azure Backup Protected VM |
| `azurerm_bastion_host` | LIVE | Azure Bastion |
| `azurerm_batch_pool` | LIVE | Azure Batch Pool |
| `azurerm_cdn_endpoint` | LIVE | Azure CDN Endpoint |
| `azurerm_cdn_frontdoor_custom_domain` | FREE | azurerm_cdn_frontdoor_custom_domain |
| `azurerm_cdn_frontdoor_custom_domain_association` | FREE | azurerm_cdn_frontdoor_custom_domain_association |
| `azurerm_cdn_frontdoor_endpoint` | FREE | azurerm_cdn_frontdoor_endpoint |
| `azurerm_cdn_frontdoor_firewall_policy` | FREE | azurerm_cdn_frontdoor_firewall_policy |
| `azurerm_cdn_frontdoor_origin` | FREE | azurerm_cdn_frontdoor_origin |
| `azurerm_cdn_frontdoor_origin_group` | FREE | azurerm_cdn_frontdoor_origin_group |
| `azurerm_cdn_frontdoor_profile` | LIVE | Azure Front Door (Standard/Premium) Profile |
| `azurerm_cdn_frontdoor_route` | FREE | azurerm_cdn_frontdoor_route |
| `azurerm_cdn_frontdoor_rule` | FREE | azurerm_cdn_frontdoor_rule |
| `azurerm_cdn_frontdoor_rule_set` | FREE | azurerm_cdn_frontdoor_rule_set |
| `azurerm_cdn_frontdoor_secret` | FREE | azurerm_cdn_frontdoor_secret |
| `azurerm_cdn_frontdoor_security_policy` | FREE | azurerm_cdn_frontdoor_security_policy |
| `azurerm_cognitive_account` | STATIC | Azure AI Services Account |
| `azurerm_container_app` | LIVE | Azure Container App |
| `azurerm_container_app_custom_domain` | FREE | azurerm_container_app_custom_domain |
| `azurerm_container_app_environment_certificate` | FREE | azurerm_container_app_environment_certificate |
| `azurerm_container_app_environment_custom_domain` | FREE | azurerm_container_app_environment_custom_domain |
| `azurerm_container_app_environment_dapr_component` | FREE | azurerm_container_app_environment_dapr_component |
| `azurerm_container_app_environment_storage` | FREE | azurerm_container_app_environment_storage |
| `azurerm_container_group` | LIVE | Azure Container Instances |
| `azurerm_container_registry` | LIVE | Azure Container Registry |
| `azurerm_container_registry_cache_rule` | FREE | azurerm_container_registry_cache_rule |
| `azurerm_container_registry_scope_map` | FREE | azurerm_container_registry_scope_map |
| `azurerm_container_registry_token` | FREE | azurerm_container_registry_token |
| `azurerm_container_registry_token_password` | FREE | azurerm_container_registry_token_password |
| `azurerm_container_registry_webhook` | FREE | azurerm_container_registry_webhook |
| `azurerm_cosmosdb_account` | LIVE | Azure Cosmos DB Account |
| `azurerm_cosmosdb_cassandra_cluster` | LIVE | Azure Managed Cassandra Cluster |
| `azurerm_cosmosdb_cassandra_keyspace` | FREE | azurerm_cosmosdb_cassandra_keyspace |
| `azurerm_cosmosdb_cassandra_table` | FREE | azurerm_cosmosdb_cassandra_table |
| `azurerm_cosmosdb_gremlin_database` | FREE | azurerm_cosmosdb_gremlin_database |
| `azurerm_cosmosdb_gremlin_graph` | FREE | azurerm_cosmosdb_gremlin_graph |
| `azurerm_cosmosdb_mongo_collection` | FREE | azurerm_cosmosdb_mongo_collection |
| `azurerm_cosmosdb_mongo_database` | FREE | azurerm_cosmosdb_mongo_database |
| `azurerm_cosmosdb_sql_container` | FREE | azurerm_cosmosdb_sql_container |
| `azurerm_cosmosdb_sql_database` | FREE | azurerm_cosmosdb_sql_database |
| `azurerm_cosmosdb_sql_function` | FREE | azurerm_cosmosdb_sql_function |
| `azurerm_cosmosdb_sql_stored_procedure` | FREE | azurerm_cosmosdb_sql_stored_procedure |
| `azurerm_cosmosdb_sql_trigger` | FREE | azurerm_cosmosdb_sql_trigger |
| `azurerm_cosmosdb_table` | FREE | azurerm_cosmosdb_table |
| `azurerm_dashboard_grafana` | LIVE | Azure Managed Grafana |
| `azurerm_dashboard_grafana_managed_private_endpoint` | FREE | azurerm_dashboard_grafana_managed_private_endpoint |
| `azurerm_data_factory` | STATIC | Azure Data Factory |
| `azurerm_data_factory_dataset_azure_blob` | FREE | azurerm_data_factory_dataset_azure_blob |
| `azurerm_data_factory_dataset_binary` | FREE | azurerm_data_factory_dataset_binary |
| `azurerm_data_factory_dataset_delimited_text` | FREE | azurerm_data_factory_dataset_delimited_text |
| `azurerm_data_factory_dataset_json` | FREE | azurerm_data_factory_dataset_json |
| `azurerm_data_factory_dataset_mysql` | FREE | azurerm_data_factory_dataset_mysql |
| `azurerm_data_factory_dataset_parquet` | FREE | azurerm_data_factory_dataset_parquet |
| `azurerm_data_factory_dataset_postgresql` | FREE | azurerm_data_factory_dataset_postgresql |
| `azurerm_data_factory_dataset_sql_server_table` | FREE | azurerm_data_factory_dataset_sql_server_table |
| `azurerm_data_factory_integration_runtime_azure` | FREE | azurerm_data_factory_integration_runtime_azure |
| `azurerm_data_factory_integration_runtime_self_hosted` | FREE | azurerm_data_factory_integration_runtime_self_hosted |
| `azurerm_data_factory_linked_service_azure_blob_storage` | FREE | azurerm_data_factory_linked_service_azure_blob_storage |
| `azurerm_data_factory_linked_service_azure_databricks` | FREE | azurerm_data_factory_linked_service_azure_databricks |
| `azurerm_data_factory_linked_service_azure_file_storage` | FREE | azurerm_data_factory_linked_service_azure_file_storage |
| `azurerm_data_factory_linked_service_azure_function` | FREE | azurerm_data_factory_linked_service_azure_function |
| `azurerm_data_factory_linked_service_azure_sql_database` | FREE | azurerm_data_factory_linked_service_azure_sql_database |
| `azurerm_data_factory_linked_service_azure_table_storage` | FREE | azurerm_data_factory_linked_service_azure_table_storage |
| `azurerm_data_factory_linked_service_cosmosdb` | FREE | azurerm_data_factory_linked_service_cosmosdb |
| `azurerm_data_factory_linked_service_data_lake_storage_gen2` | FREE | azurerm_data_factory_linked_service_data_lake_storage_gen2 |
| `azurerm_data_factory_linked_service_key_vault` | FREE | azurerm_data_factory_linked_service_key_vault |
| `azurerm_data_factory_linked_service_mysql` | FREE | azurerm_data_factory_linked_service_mysql |
| `azurerm_data_factory_linked_service_postgresql` | FREE | azurerm_data_factory_linked_service_postgresql |
| `azurerm_data_factory_linked_service_sftp` | FREE | azurerm_data_factory_linked_service_sftp |
| `azurerm_data_factory_linked_service_snowflake` | FREE | azurerm_data_factory_linked_service_snowflake |
| `azurerm_data_factory_linked_service_sql_server` | FREE | azurerm_data_factory_linked_service_sql_server |
| `azurerm_data_factory_linked_service_synapse` | FREE | azurerm_data_factory_linked_service_synapse |
| `azurerm_data_factory_linked_service_web` | FREE | azurerm_data_factory_linked_service_web |
| `azurerm_data_factory_managed_private_endpoint` | FREE | azurerm_data_factory_managed_private_endpoint |
| `azurerm_data_factory_pipeline` | FREE | azurerm_data_factory_pipeline |
| `azurerm_data_factory_trigger_blob_event` | FREE | azurerm_data_factory_trigger_blob_event |
| `azurerm_data_factory_trigger_schedule` | FREE | azurerm_data_factory_trigger_schedule |
| `azurerm_data_lake_storage_gen2_filesystem` | FREE | Azure Data Lake Storage Gen2 Filesystem |
| `azurerm_data_protection_backup_vault` | STATIC | Azure Backup Vault (Data Protection) |
| `azurerm_databricks_workspace` | LIVE | Azure Databricks Workspace |
| `azurerm_ddos_protection_plan` | LIVE | Azure DDoS Network Protection Plan |
| `azurerm_dedicated_host` | STATIC | Azure Dedicated Host |
| `azurerm_dev_test_lab` | FREE | azurerm_dev_test_lab |
| `azurerm_dev_test_policy` | FREE | azurerm_dev_test_policy |
| `azurerm_dev_test_schedule` | FREE | azurerm_dev_test_schedule |
| `azurerm_digital_twins_instance` | LIVE | Azure Digital Twins Instance |
| `azurerm_dns_a_record` | FREE | azurerm_dns_a_record |
| `azurerm_dns_aaaa_record` | FREE | azurerm_dns_aaaa_record |
| `azurerm_dns_caa_record` | FREE | azurerm_dns_caa_record |
| `azurerm_dns_cname_record` | FREE | azurerm_dns_cname_record |
| `azurerm_dns_mx_record` | FREE | azurerm_dns_mx_record |
| `azurerm_dns_ns_record` | FREE | azurerm_dns_ns_record |
| `azurerm_dns_ptr_record` | FREE | azurerm_dns_ptr_record |
| `azurerm_dns_srv_record` | FREE | azurerm_dns_srv_record |
| `azurerm_dns_txt_record` | FREE | azurerm_dns_txt_record |
| `azurerm_dns_zone` | STATIC | Azure DNS Zone |
| `azurerm_eventgrid_domain` | LIVE | Azure Event Grid Domain |
| `azurerm_eventgrid_domain_topic` | FREE | azurerm_eventgrid_domain_topic |
| `azurerm_eventgrid_event_subscription` | FREE | Azure Eventgrid Event Subscription |
| `azurerm_eventgrid_system_topic` | FREE | azurerm_eventgrid_system_topic |
| `azurerm_eventgrid_system_topic_event_subscription` | FREE | azurerm_eventgrid_system_topic_event_subscription |
| `azurerm_eventgrid_topic` | LIVE | Azure Event Grid Topic |
| `azurerm_eventhub` | FREE | Azure Eventhub |
| `azurerm_eventhub_authorization_rule` | FREE | Azure Eventhub Authorization Rule |
| `azurerm_eventhub_cluster` | STATIC | Azure Event Hubs Dedicated Cluster |
| `azurerm_eventhub_consumer_group` | FREE | Azure Eventhub Consumer Group |
| `azurerm_eventhub_namespace` | LIVE | Azure Event Hubs Namespace |
| `azurerm_eventhub_namespace_authorization_rule` | FREE | azurerm_eventhub_namespace_authorization_rule |
| `azurerm_express_route_circuit` | LIVE | Azure ExpressRoute Circuit |
| `azurerm_federated_identity_credential` | FREE | azurerm_federated_identity_credential |
| `azurerm_firewall` | STATIC | Azure Firewall |
| `azurerm_firewall_application_rule_collection` | FREE | azurerm_firewall_application_rule_collection |
| `azurerm_firewall_nat_rule_collection` | FREE | azurerm_firewall_nat_rule_collection |
| `azurerm_firewall_network_rule_collection` | FREE | azurerm_firewall_network_rule_collection |
| `azurerm_firewall_policy` | FREE | azurerm_firewall_policy |
| `azurerm_firewall_policy_rule_collection_group` | FREE | azurerm_firewall_policy_rule_collection_group |
| `azurerm_function_app_function` | FREE | Azure Function App Function |
| `azurerm_function_app_hybrid_connection` | FREE | azurerm_function_app_hybrid_connection |
| `azurerm_hdinsight_hadoop_cluster` | STATIC | Azure HDInsight Cluster |
| `azurerm_hdinsight_hbase_cluster` | STATIC | Azure HDInsight Cluster |
| `azurerm_hdinsight_interactive_query_cluster` | STATIC | Azure HDInsight Cluster |
| `azurerm_hdinsight_kafka_cluster` | STATIC | Azure HDInsight Cluster |
| `azurerm_hdinsight_spark_cluster` | STATIC | Azure HDInsight Cluster |
| `azurerm_healthcare_fhir_service` | LIVE | Azure Health Data FHIR Service |
| `azurerm_image` | STATIC | Azure Managed VM Image |
| `azurerm_iothub` | LIVE | Azure IoT Hub |
| `azurerm_iothub_consumer_group` | FREE | azurerm_iothub_consumer_group |
| `azurerm_iothub_dps` | STATIC | Azure IoT Hub DPS |
| `azurerm_iothub_endpoint_eventhub` | FREE | azurerm_iothub_endpoint_eventhub |
| `azurerm_iothub_endpoint_servicebus_queue` | FREE | azurerm_iothub_endpoint_servicebus_queue |
| `azurerm_iothub_endpoint_servicebus_topic` | FREE | azurerm_iothub_endpoint_servicebus_topic |
| `azurerm_iothub_endpoint_storage_container` | FREE | azurerm_iothub_endpoint_storage_container |
| `azurerm_iothub_enrichment` | FREE | azurerm_iothub_enrichment |
| `azurerm_iothub_route` | FREE | azurerm_iothub_route |
| `azurerm_iothub_shared_access_policy` | FREE | azurerm_iothub_shared_access_policy |
| `azurerm_ip_group` | FREE | azurerm_ip_group |
| `azurerm_key_vault` | LIVE | Azure Key Vault |
| `azurerm_key_vault_access_policy` | FREE | Azure Key Vault Access Policy |
| `azurerm_key_vault_certificate` | FREE | Azure Key Vault Certificate |
| `azurerm_key_vault_certificate_contacts` | FREE | azurerm_key_vault_certificate_contacts |
| `azurerm_key_vault_certificate_issuer` | FREE | azurerm_key_vault_certificate_issuer |
| `azurerm_key_vault_key` | FREE | azurerm_key_vault_key |
| `azurerm_key_vault_managed_storage_account` | FREE | azurerm_key_vault_managed_storage_account |
| `azurerm_key_vault_secret` | FREE | Azure Key Vault Secret |
| `azurerm_kubernetes_cluster` | LIVE | Azure Kubernetes Service Cluster |
| `azurerm_kubernetes_cluster_extension` | FREE | azurerm_kubernetes_cluster_extension |
| `azurerm_kubernetes_cluster_node_pool` | LIVE | AKS Additional Node Pool |
| `azurerm_kubernetes_flux_configuration` | FREE | azurerm_kubernetes_flux_configuration |
| `azurerm_kusto_cluster` | STATIC | Azure Data Explorer Cluster |
| `azurerm_lb` | STATIC | Azure Load Balancer |
| `azurerm_lb_backend_address_pool` | FREE | Azure Lb Backend Address Pool |
| `azurerm_lb_backend_address_pool_address` | FREE | azurerm_lb_backend_address_pool_address |
| `azurerm_lb_nat_pool` | FREE | azurerm_lb_nat_pool |
| `azurerm_lb_nat_rule` | FREE | Azure Lb Nat Rule |
| `azurerm_lb_outbound_rule` | FREE | azurerm_lb_outbound_rule |
| `azurerm_lb_probe` | FREE | Azure Lb Probe |
| `azurerm_lb_rule` | FREE | Azure Lb Rule |
| `azurerm_linux_function_app` | STATIC | Azure Functions App |
| `azurerm_linux_function_app_slot` | FREE | azurerm_linux_function_app_slot |
| `azurerm_linux_virtual_machine` | LIVE | Azure Linux Virtual Machine |
| `azurerm_linux_virtual_machine_scale_set` | STATIC | Azure Linux VM Scale Set |
| `azurerm_linux_web_app_slot` | FREE | azurerm_linux_web_app_slot |
| `azurerm_load_test` | STATIC | Azure Load Testing Resource |
| `azurerm_local_network_gateway` | FREE | azurerm_local_network_gateway |
| `azurerm_log_analytics_cluster` | STATIC | Azure Log Analytics Dedicated Cluster |
| `azurerm_log_analytics_data_export_rule` | FREE | azurerm_log_analytics_data_export_rule |
| `azurerm_log_analytics_datasource_windows_event` | FREE | azurerm_log_analytics_datasource_windows_event |
| `azurerm_log_analytics_datasource_windows_performance_counter` | FREE | azurerm_log_analytics_datasource_windows_performance_counter |
| `azurerm_log_analytics_linked_service` | FREE | azurerm_log_analytics_linked_service |
| `azurerm_log_analytics_linked_storage_account` | FREE | azurerm_log_analytics_linked_storage_account |
| `azurerm_log_analytics_query_pack` | FREE | azurerm_log_analytics_query_pack |
| `azurerm_log_analytics_query_pack_query` | FREE | azurerm_log_analytics_query_pack_query |
| `azurerm_log_analytics_saved_search` | FREE | azurerm_log_analytics_saved_search |
| `azurerm_log_analytics_solution` | FREE | Azure Log Analytics Solution |
| `azurerm_log_analytics_storage_insights` | FREE | azurerm_log_analytics_storage_insights |
| `azurerm_log_analytics_workspace` | STATIC | Azure Log Analytics Workspace |
| `azurerm_logic_app_standard` | STATIC | Azure Logic App (Standard) |
| `azurerm_logic_app_workflow` | LIVE | Azure Logic App (Consumption) |
| `azurerm_machine_learning_compute_cluster` | LIVE | Azure ML Compute Cluster |
| `azurerm_machine_learning_compute_instance` | LIVE | Azure ML Compute Instance |
| `azurerm_maintenance_assignment_virtual_machine` | FREE | azurerm_maintenance_assignment_virtual_machine |
| `azurerm_maintenance_configuration` | FREE | azurerm_maintenance_configuration |
| `azurerm_managed_disk` | LIVE | Azure Managed Disk |
| `azurerm_managed_lustre_file_system` | LIVE | Azure Managed Lustre File System |
| `azurerm_management_group` | FREE | azurerm_management_group |
| `azurerm_management_group_policy_assignment` | FREE | azurerm_management_group_policy_assignment |
| `azurerm_management_group_subscription_association` | FREE | azurerm_management_group_subscription_association |
| `azurerm_management_group_template_deployment` | FREE | azurerm_management_group_template_deployment |
| `azurerm_management_lock` | FREE | azurerm_management_lock |
| `azurerm_maps_account` | LIVE | Azure Maps Account |
| `azurerm_mariadb_server` | STATIC | Azure Database for MariaDB |
| `azurerm_marketplace_agreement` | FREE | azurerm_marketplace_agreement |
| `azurerm_monitor_action_group` | FREE | azurerm_monitor_action_group |
| `azurerm_monitor_activity_log_alert` | FREE | azurerm_monitor_activity_log_alert |
| `azurerm_monitor_autoscale_setting` | FREE | azurerm_monitor_autoscale_setting |
| `azurerm_monitor_data_collection_endpoint` | FREE | azurerm_monitor_data_collection_endpoint |
| `azurerm_monitor_data_collection_rule` | FREE | azurerm_monitor_data_collection_rule |
| `azurerm_monitor_data_collection_rule_association` | FREE | azurerm_monitor_data_collection_rule_association |
| `azurerm_monitor_diagnostic_setting` | FREE | azurerm_monitor_diagnostic_setting |
| `azurerm_monitor_metric_alert` | LIVE | Azure Monitor Metric Alert |
| `azurerm_monitor_private_link_scope` | FREE | azurerm_monitor_private_link_scope |
| `azurerm_monitor_private_link_scoped_service` | FREE | azurerm_monitor_private_link_scoped_service |
| `azurerm_mssql_database` | STATIC | Azure SQL Database |
| `azurerm_mssql_database_extended_auditing_policy` | FREE | azurerm_mssql_database_extended_auditing_policy |
| `azurerm_mssql_database_vulnerability_assessment_rule_baseline` | FREE | azurerm_mssql_database_vulnerability_assessment_rule_baseline |
| `azurerm_mssql_elasticpool` | STATIC | Azure SQL Elastic Pool |
| `azurerm_mssql_firewall_rule` | FREE | azurerm_mssql_firewall_rule |
| `azurerm_mssql_managed_instance` | STATIC | Azure SQL Managed Instance |
| `azurerm_mssql_outbound_firewall_rule` | FREE | azurerm_mssql_outbound_firewall_rule |
| `azurerm_mssql_server` | FREE | Azure Mssql Server |
| `azurerm_mssql_server_dns_alias` | FREE | azurerm_mssql_server_dns_alias |
| `azurerm_mssql_server_extended_auditing_policy` | FREE | azurerm_mssql_server_extended_auditing_policy |
| `azurerm_mssql_server_microsoft_support_auditing_policy` | FREE | azurerm_mssql_server_microsoft_support_auditing_policy |
| `azurerm_mssql_server_security_alert_policy` | FREE | azurerm_mssql_server_security_alert_policy |
| `azurerm_mssql_server_transparent_data_encryption` | FREE | azurerm_mssql_server_transparent_data_encryption |
| `azurerm_mssql_virtual_network_rule` | FREE | azurerm_mssql_virtual_network_rule |
| `azurerm_mysql_flexible_database` | FREE | azurerm_mysql_flexible_database |
| `azurerm_mysql_flexible_server` | STATIC | Azure Database for MySQL Flexible Server |
| `azurerm_mysql_flexible_server_configuration` | FREE | azurerm_mysql_flexible_server_configuration |
| `azurerm_mysql_flexible_server_firewall_rule` | FREE | azurerm_mysql_flexible_server_firewall_rule |
| `azurerm_mysql_server` | STATIC | Azure MySQL Single Server (legacy) |
| `azurerm_nat_gateway` | LIVE | Azure NAT Gateway |
| `azurerm_nat_gateway_public_ip_association` | FREE | azurerm_nat_gateway_public_ip_association |
| `azurerm_nat_gateway_public_ip_prefix_association` | FREE | azurerm_nat_gateway_public_ip_prefix_association |
| `azurerm_netapp_pool` | LIVE | Azure NetApp Files Capacity Pool |
| `azurerm_network_interface` | FREE | Azure Network Interface |
| `azurerm_network_interface_application_gateway_backend_address_pool_association` | FREE | azurerm_network_interface_application_gateway_backend_address_pool_association |
| `azurerm_network_interface_application_security_group_association` | FREE | azurerm_network_interface_application_security_group_association |
| `azurerm_network_interface_backend_address_pool_association` | FREE | azurerm_network_interface_backend_address_pool_association |
| `azurerm_network_interface_nat_rule_association` | FREE | azurerm_network_interface_nat_rule_association |
| `azurerm_network_interface_security_group_association` | FREE | azurerm_network_interface_security_group_association |
| `azurerm_network_security_group` | FREE | Azure Network Security Group |
| `azurerm_network_security_rule` | FREE | azurerm_network_security_rule |
| `azurerm_notification_hub_authorization_rule` | FREE | azurerm_notification_hub_authorization_rule |
| `azurerm_orchestrated_virtual_machine_scale_set` | LIVE | Azure Orchestrated VM Scale Set |
| `azurerm_point_to_site_vpn_gateway` | STATIC | Azure vWAN P2S VPN Gateway |
| `azurerm_policy_definition` | FREE | azurerm_policy_definition |
| `azurerm_policy_set_definition` | FREE | azurerm_policy_set_definition |
| `azurerm_policy_virtual_machine_configuration_assignment` | FREE | azurerm_policy_virtual_machine_configuration_assignment |
| `azurerm_portal_dashboard` | FREE | azurerm_portal_dashboard |
| `azurerm_postgresql_flexible_server` | LIVE | Azure Database for PostgreSQL Flexible Server |
| `azurerm_postgresql_flexible_server_active_directory_administrator` | FREE | azurerm_postgresql_flexible_server_active_directory_administrator |
| `azurerm_postgresql_flexible_server_configuration` | FREE | Azure Postgresql Flexible Server Configuration |
| `azurerm_postgresql_flexible_server_database` | FREE | Azure Postgresql Flexible Server Database |
| `azurerm_postgresql_flexible_server_firewall_rule` | FREE | Azure Postgresql Flexible Server Firewall Rule |
| `azurerm_postgresql_flexible_server_virtual_endpoint` | FREE | azurerm_postgresql_flexible_server_virtual_endpoint |
| `azurerm_postgresql_server` | STATIC | Azure PostgreSQL Single Server (legacy) |
| `azurerm_powerbi_embedded` | LIVE | Power BI Embedded Capacity |
| `azurerm_private_dns_a_record` | FREE | Azure Private Dns A Record |
| `azurerm_private_dns_aaaa_record` | FREE | azurerm_private_dns_aaaa_record |
| `azurerm_private_dns_cname_record` | FREE | azurerm_private_dns_cname_record |
| `azurerm_private_dns_mx_record` | FREE | azurerm_private_dns_mx_record |
| `azurerm_private_dns_ptr_record` | FREE | azurerm_private_dns_ptr_record |
| `azurerm_private_dns_resolver` | STATIC | Azure DNS Private Resolver |
| `azurerm_private_dns_srv_record` | FREE | azurerm_private_dns_srv_record |
| `azurerm_private_dns_txt_record` | FREE | azurerm_private_dns_txt_record |
| `azurerm_private_dns_zone` | STATIC | Azure Private DNS Zone |
| `azurerm_private_dns_zone_virtual_network_link` | FREE | azurerm_private_dns_zone_virtual_network_link |
| `azurerm_private_endpoint` | STATIC | Azure Private Endpoint |
| `azurerm_private_endpoint_application_security_group_association` | FREE | azurerm_private_endpoint_application_security_group_association |
| `azurerm_private_link_service` | FREE | azurerm_private_link_service |
| `azurerm_public_ip` | LIVE | Azure Public IP Address |
| `azurerm_purview_account` | LIVE | Microsoft Purview Account |
| `azurerm_recovery_services_vault` | FREE | Azure Recovery Services Vault |
| `azurerm_redis_cache` | LIVE | Azure Cache for Redis |
| `azurerm_redis_enterprise_cluster` | LIVE | Azure Cache for Redis Enterprise |
| `azurerm_redis_firewall_rule` | FREE | Azure Redis Firewall Rule |
| `azurerm_redis_linked_server` | FREE | azurerm_redis_linked_server |
| `azurerm_relay_namespace` | STATIC | Azure Relay Namespace |
| `azurerm_resource_group` | FREE | Azure Resource Group |
| `azurerm_resource_group_policy_assignment` | FREE | azurerm_resource_group_policy_assignment |
| `azurerm_resource_group_template_deployment` | FREE | azurerm_resource_group_template_deployment |
| `azurerm_resource_policy_assignment` | FREE | azurerm_resource_policy_assignment |
| `azurerm_resource_provider_registration` | FREE | azurerm_resource_provider_registration |
| `azurerm_role_assignment` | FREE | Azure Role Assignment |
| `azurerm_role_definition` | FREE | azurerm_role_definition |
| `azurerm_role_management_policy` | FREE | azurerm_role_management_policy |
| `azurerm_route` | FREE | Azure Route |
| `azurerm_route_filter` | FREE | azurerm_route_filter |
| `azurerm_route_server_bgp_connection` | FREE | azurerm_route_server_bgp_connection |
| `azurerm_route_table` | FREE | Azure Route Table |
| `azurerm_search_service` | LIVE | Azure AI Search |
| `azurerm_sentinel_log_analytics_workspace_onboarding` | LIVE | Microsoft Sentinel Workspace |
| `azurerm_service_plan` | LIVE | Azure App Service Plan |
| `azurerm_servicebus_namespace` | STATIC | Azure Service Bus Namespace |
| `azurerm_servicebus_namespace_authorization_rule` | FREE | azurerm_servicebus_namespace_authorization_rule |
| `azurerm_servicebus_queue` | FREE | Azure Servicebus Queue |
| `azurerm_servicebus_queue_authorization_rule` | FREE | azurerm_servicebus_queue_authorization_rule |
| `azurerm_servicebus_subscription` | FREE | Azure Servicebus Subscription |
| `azurerm_servicebus_subscription_rule` | FREE | azurerm_servicebus_subscription_rule |
| `azurerm_servicebus_topic` | FREE | Azure Servicebus Topic |
| `azurerm_servicebus_topic_authorization_rule` | FREE | azurerm_servicebus_topic_authorization_rule |
| `azurerm_signalr_service` | LIVE | Azure SignalR Service |
| `azurerm_site_recovery_replicated_vm` | LIVE | Azure Site Recovery Replicated VM |
| `azurerm_snapshot` | STATIC | Azure Managed Disk Snapshot |
| `azurerm_spring_cloud_service` | STATIC | Azure Spring Apps Service |
| `azurerm_static_web_app` | STATIC | Azure Static Web App |
| `azurerm_static_web_app_custom_domain` | FREE | azurerm_static_web_app_custom_domain |
| `azurerm_storage_account` | LIVE | Azure Storage Account |
| `azurerm_storage_account_customer_managed_key` | FREE | azurerm_storage_account_customer_managed_key |
| `azurerm_storage_account_local_user` | FREE | azurerm_storage_account_local_user |
| `azurerm_storage_account_network_rules` | FREE | azurerm_storage_account_network_rules |
| `azurerm_storage_blob` | FREE | Azure Storage Blob |
| `azurerm_storage_container` | FREE | Azure Storage Container |
| `azurerm_storage_data_lake_gen2_path` | FREE | azurerm_storage_data_lake_gen2_path |
| `azurerm_storage_encryption_scope` | FREE | azurerm_storage_encryption_scope |
| `azurerm_storage_management_policy` | FREE | Azure Storage Management Policy |
| `azurerm_storage_object_replication` | FREE | azurerm_storage_object_replication |
| `azurerm_storage_queue` | FREE | Azure Storage Queue |
| `azurerm_storage_share` | LIVE | Azure Files Share |
| `azurerm_storage_share_directory` | FREE | azurerm_storage_share_directory |
| `azurerm_storage_table` | FREE | Azure Storage Table |
| `azurerm_storage_table_entity` | FREE | azurerm_storage_table_entity |
| `azurerm_stream_analytics_job` | LIVE | Azure Stream Analytics Job |
| `azurerm_subnet` | FREE | Azure Subnet |
| `azurerm_subnet_nat_gateway_association` | FREE | azurerm_subnet_nat_gateway_association |
| `azurerm_subnet_network_security_group_association` | FREE | azurerm_subnet_network_security_group_association |
| `azurerm_subnet_route_table_association` | FREE | azurerm_subnet_route_table_association |
| `azurerm_subnet_service_endpoint_storage_policy` | FREE | azurerm_subnet_service_endpoint_storage_policy |
| `azurerm_subscription` | FREE | azurerm_subscription |
| `azurerm_subscription_policy_assignment` | FREE | azurerm_subscription_policy_assignment |
| `azurerm_subscription_template_deployment` | FREE | azurerm_subscription_template_deployment |
| `azurerm_synapse_firewall_rule` | FREE | azurerm_synapse_firewall_rule |
| `azurerm_synapse_integration_runtime_azure` | FREE | azurerm_synapse_integration_runtime_azure |
| `azurerm_synapse_integration_runtime_self_hosted` | FREE | azurerm_synapse_integration_runtime_self_hosted |
| `azurerm_synapse_linked_service` | FREE | azurerm_synapse_linked_service |
| `azurerm_synapse_managed_private_endpoint` | FREE | azurerm_synapse_managed_private_endpoint |
| `azurerm_synapse_private_link_hub` | FREE | azurerm_synapse_private_link_hub |
| `azurerm_synapse_role_assignment` | FREE | azurerm_synapse_role_assignment |
| `azurerm_synapse_spark_pool` | STATIC | Azure Synapse Spark Pool |
| `azurerm_synapse_spark_pool_workspace_attachment_disabled_placeholder` | FREE | azurerm_synapse_spark_pool_workspace_attachment_disabled_placeholder |
| `azurerm_synapse_sql_pool` | STATIC | Azure Synapse Dedicated SQL Pool |
| `azurerm_synapse_workspace` | LIVE | Azure Synapse Workspace |
| `azurerm_tenant_template_deployment` | FREE | azurerm_tenant_template_deployment |
| `azurerm_traffic_manager_profile` | LIVE | Azure Traffic Manager Profile |
| `azurerm_user_assigned_identity` | FREE | Azure User Assigned Identity |
| `azurerm_virtual_hub` | STATIC | Azure Virtual WAN Hub |
| `azurerm_virtual_machine` | LIVE | Azure Virtual Machine (legacy kind) |
| `azurerm_virtual_network` | FREE | Azure Virtual Network |
| `azurerm_virtual_network_dns_servers` | FREE | azurerm_virtual_network_dns_servers |
| `azurerm_virtual_network_gateway` | LIVE | Azure VPN Gateway |
| `azurerm_virtual_network_gateway_connection` | FREE | azurerm_virtual_network_gateway_connection |
| `azurerm_virtual_network_peering` | FREE | azurerm_virtual_network_peering |
| `azurerm_vpn_gateway` | STATIC | Azure vWAN S2S VPN Gateway |
| `azurerm_web_app_active_slot` | FREE | azurerm_web_app_active_slot |
| `azurerm_web_app_hybrid_connection` | FREE | azurerm_web_app_hybrid_connection |
| `azurerm_web_application_firewall_policy` | FREE | azurerm_web_application_firewall_policy |
| `azurerm_web_pubsub` | LIVE | Azure Web PubSub Service |
| `azurerm_windows_function_app_slot` | FREE | azurerm_windows_function_app_slot |
| `azurerm_windows_virtual_machine` | LIVE | Azure Windows Virtual Machine |
| `azurerm_windows_virtual_machine_scale_set` | STATIC | Azure Windows VM Scale Set |
| `azurerm_windows_web_app_slot` | FREE | azurerm_windows_web_app_slot |

**AZURE totals:** 57 LIVE · 46 STATIC · 306 FREE

## GCP (306 resources)

| Kind | Status | Display name |
|---|---|---|
| `google_alloydb_cluster` | STATIC | GCP AlloyDB Cluster |
| `google_alloydb_instance` | STATIC | GCP AlloyDB Instance |
| `google_api_gateway_api` | STATIC | GCP API Gateway API |
| `google_app_engine_application` | FREE | google_app_engine_application |
| `google_app_engine_application_url_dispatch_rules` | FREE | google_app_engine_application_url_dispatch_rules |
| `google_app_engine_domain_mapping` | FREE | google_app_engine_domain_mapping |
| `google_app_engine_firewall_rule` | FREE | google_app_engine_firewall_rule |
| `google_app_engine_service_network_settings` | FREE | google_app_engine_service_network_settings |
| `google_app_engine_service_split_traffic` | FREE | google_app_engine_service_split_traffic |
| `google_artifact_registry_repository` | LIVE | GCP Artifact Registry Repository |
| `google_artifact_registry_repository_iam_binding` | FREE | google_artifact_registry_repository_iam_binding |
| `google_artifact_registry_repository_iam_member` | FREE | google_artifact_registry_repository_iam_member |
| `google_artifact_registry_repository_iam_policy` | FREE | google_artifact_registry_repository_iam_policy |
| `google_bigquery_data_transfer_config` | FREE | google_bigquery_data_transfer_config |
| `google_bigquery_dataset` | LIVE | GCP BigQuery Dataset |
| `google_bigquery_dataset_access` | FREE | google_bigquery_dataset_access |
| `google_bigquery_dataset_iam_binding` | FREE | google_bigquery_dataset_iam_binding |
| `google_bigquery_dataset_iam_member` | FREE | google_bigquery_dataset_iam_member |
| `google_bigquery_dataset_iam_policy` | FREE | google_bigquery_dataset_iam_policy |
| `google_bigquery_job` | FREE | GCP Bigquery Job |
| `google_bigquery_reservation` | STATIC | GCP BigQuery Slot Reservation |
| `google_bigquery_reservation_assignment` | FREE | google_bigquery_reservation_assignment |
| `google_bigquery_routine` | FREE | google_bigquery_routine |
| `google_bigquery_table` | FREE | GCP Bigquery Table |
| `google_bigquery_table_iam_binding` | FREE | google_bigquery_table_iam_binding |
| `google_bigquery_table_iam_member` | FREE | google_bigquery_table_iam_member |
| `google_bigquery_table_iam_policy` | FREE | google_bigquery_table_iam_policy |
| `google_bigtable_app_profile` | FREE | google_bigtable_app_profile |
| `google_bigtable_gc_policy` | FREE | google_bigtable_gc_policy |
| `google_bigtable_instance` | STATIC | GCP Bigtable Instance |
| `google_bigtable_instance_iam_binding` | FREE | google_bigtable_instance_iam_binding |
| `google_bigtable_instance_iam_member` | FREE | google_bigtable_instance_iam_member |
| `google_bigtable_instance_iam_policy` | FREE | google_bigtable_instance_iam_policy |
| `google_bigtable_table` | FREE | google_bigtable_table |
| `google_bigtable_table_iam_binding` | FREE | google_bigtable_table_iam_binding |
| `google_bigtable_table_iam_member` | FREE | google_bigtable_table_iam_member |
| `google_bigtable_table_iam_policy` | FREE | google_bigtable_table_iam_policy |
| `google_billing_account_iam_binding` | FREE | google_billing_account_iam_binding |
| `google_billing_account_iam_member` | FREE | google_billing_account_iam_member |
| `google_billing_account_iam_policy` | FREE | google_billing_account_iam_policy |
| `google_billing_budget` | FREE | google_billing_budget |
| `google_certificate_manager_certificate` | FREE | GCP Certificate Manager Certificate |
| `google_cloud_ids_endpoint` | STATIC | GCP Cloud IDS Endpoint |
| `google_cloud_run_domain_mapping` | FREE | google_cloud_run_domain_mapping |
| `google_cloud_run_service` | LIVE | GCP Cloud Run Service |
| `google_cloud_run_service_iam_binding` | FREE | google_cloud_run_service_iam_binding |
| `google_cloud_run_service_iam_member` | FREE | GCP Cloud Run Service Iam Member |
| `google_cloud_run_service_iam_policy` | FREE | google_cloud_run_service_iam_policy |
| `google_cloud_run_v2_job` | FREE | google_cloud_run_v2_job |
| `google_cloud_run_v2_job_iam_binding` | FREE | google_cloud_run_v2_job_iam_binding |
| `google_cloud_run_v2_job_iam_member` | FREE | google_cloud_run_v2_job_iam_member |
| `google_cloud_run_v2_job_iam_policy` | FREE | google_cloud_run_v2_job_iam_policy |
| `google_cloud_run_v2_service` | LIVE | GCP Cloud Run Service (v2) |
| `google_cloud_run_v2_service_iam_binding` | FREE | google_cloud_run_v2_service_iam_binding |
| `google_cloud_run_v2_service_iam_member` | FREE | google_cloud_run_v2_service_iam_member |
| `google_cloud_run_v2_service_iam_policy` | FREE | google_cloud_run_v2_service_iam_policy |
| `google_cloud_scheduler_job` | STATIC | GCP Cloud Scheduler Job |
| `google_cloud_tasks_queue` | STATIC | GCP Cloud Tasks Queue |
| `google_cloudbuild_trigger` | FREE | google_cloudbuild_trigger |
| `google_cloudbuild_worker_pool_disabled_placeholder` | FREE | google_cloudbuild_worker_pool_disabled_placeholder |
| `google_cloudfunctions2_function` | STATIC | GCP Cloud Functions (2nd gen) |
| `google_cloudfunctions2_function_iam_binding` | FREE | google_cloudfunctions2_function_iam_binding |
| `google_cloudfunctions2_function_iam_member` | FREE | google_cloudfunctions2_function_iam_member |
| `google_cloudfunctions2_function_iam_policy` | FREE | google_cloudfunctions2_function_iam_policy |
| `google_cloudfunctions_function` | STATIC | GCP Cloud Functions (1st gen) |
| `google_cloudfunctions_function_iam_binding` | FREE | google_cloudfunctions_function_iam_binding |
| `google_cloudfunctions_function_iam_member` | FREE | google_cloudfunctions_function_iam_member |
| `google_cloudfunctions_function_iam_policy` | FREE | google_cloudfunctions_function_iam_policy |
| `google_cloudfunctions_function_invoker` | FREE | google_cloudfunctions_function_invoker |
| `google_composer_environment` | STATIC | GCP Cloud Composer Environment |
| `google_compute_address` | LIVE | GCP Compute Static IP Address |
| `google_compute_attached_disk` | FREE | GCP Compute Attached Disk |
| `google_compute_autoscaler` | FREE | GCP Compute Autoscaler |
| `google_compute_backend_bucket` | FREE | google_compute_backend_bucket |
| `google_compute_backend_bucket_signed_url_key` | FREE | google_compute_backend_bucket_signed_url_key |
| `google_compute_backend_service` | FREE | GCP Compute Backend Service |
| `google_compute_backend_service_signed_url_key` | FREE | google_compute_backend_service_signed_url_key |
| `google_compute_disk` | LIVE | GCP Persistent Disk |
| `google_compute_disk_resource_policy_attachment` | FREE | google_compute_disk_resource_policy_attachment |
| `google_compute_firewall` | FREE | GCP Compute Firewall |
| `google_compute_firewall_policy` | FREE | google_compute_firewall_policy |
| `google_compute_firewall_policy_association` | FREE | google_compute_firewall_policy_association |
| `google_compute_firewall_policy_rule` | FREE | google_compute_firewall_policy_rule |
| `google_compute_forwarding_rule` | FREE | GCP Compute Forwarding Rule |
| `google_compute_global_address` | FREE | GCP Global IP Address |
| `google_compute_global_forwarding_rule` | FREE | GCP Compute Global Forwarding Rule |
| `google_compute_global_network_endpoint` | FREE | google_compute_global_network_endpoint |
| `google_compute_global_network_endpoint_group` | FREE | google_compute_global_network_endpoint_group |
| `google_compute_ha_vpn_gateway` | STATIC | GCP HA VPN Gateway |
| `google_compute_health_check` | FREE | GCP Compute Health Check |
| `google_compute_http_health_check` | FREE | google_compute_http_health_check |
| `google_compute_https_health_check` | FREE | google_compute_https_health_check |
| `google_compute_instance` | LIVE | GCP Compute Instance |
| `google_compute_instance_from_template` | FREE | GCP Compute Instance From Template |
| `google_compute_instance_group` | FREE | GCP Compute Instance Group |
| `google_compute_instance_group_manager` | LIVE | GCP Managed Instance Group |
| `google_compute_instance_group_membership` | FREE | google_compute_instance_group_membership |
| `google_compute_instance_group_named_port` | FREE | google_compute_instance_group_named_port |
| `google_compute_instance_iam_binding` | FREE | google_compute_instance_iam_binding |
| `google_compute_instance_iam_member` | FREE | google_compute_instance_iam_member |
| `google_compute_instance_iam_policy` | FREE | google_compute_instance_iam_policy |
| `google_compute_instance_settings` | FREE | google_compute_instance_settings |
| `google_compute_instance_template` | FREE | GCP Compute Instance Template |
| `google_compute_interconnect_attachment` | LIVE | GCP Interconnect VLAN Attachment |
| `google_compute_machine_image_iam_binding` | FREE | google_compute_machine_image_iam_binding |
| `google_compute_machine_image_iam_member` | FREE | google_compute_machine_image_iam_member |
| `google_compute_machine_image_iam_policy` | FREE | google_compute_machine_image_iam_policy |
| `google_compute_managed_ssl_certificate` | FREE | google_compute_managed_ssl_certificate |
| `google_compute_network` | FREE | GCP Compute Network |
| `google_compute_network_endpoint` | FREE | google_compute_network_endpoint |
| `google_compute_network_endpoint_group` | FREE | google_compute_network_endpoint_group |
| `google_compute_network_firewall_policy` | FREE | google_compute_network_firewall_policy |
| `google_compute_network_firewall_policy_association` | FREE | google_compute_network_firewall_policy_association |
| `google_compute_network_firewall_policy_rule` | FREE | google_compute_network_firewall_policy_rule |
| `google_compute_network_peering` | FREE | google_compute_network_peering |
| `google_compute_network_peering_routes_config` | FREE | google_compute_network_peering_routes_config |
| `google_compute_project_default_network_tier` | FREE | google_compute_project_default_network_tier |
| `google_compute_project_metadata` | FREE | google_compute_project_metadata |
| `google_compute_project_metadata_item` | FREE | google_compute_project_metadata_item |
| `google_compute_public_advertised_prefix` | FREE | google_compute_public_advertised_prefix |
| `google_compute_public_delegated_prefix` | FREE | google_compute_public_delegated_prefix |
| `google_compute_region_autoscaler` | FREE | GCP Compute Region Autoscaler |
| `google_compute_region_health_check` | FREE | google_compute_region_health_check |
| `google_compute_region_instance_group_manager` | LIVE | GCP Managed Instance Group |
| `google_compute_region_instance_template` | FREE | google_compute_region_instance_template |
| `google_compute_region_network_endpoint_group` | FREE | google_compute_region_network_endpoint_group |
| `google_compute_region_network_firewall_policy` | FREE | google_compute_region_network_firewall_policy |
| `google_compute_region_network_firewall_policy_association` | FREE | google_compute_region_network_firewall_policy_association |
| `google_compute_region_network_firewall_policy_rule` | FREE | google_compute_region_network_firewall_policy_rule |
| `google_compute_region_ssl_certificate` | FREE | google_compute_region_ssl_certificate |
| `google_compute_region_ssl_policy` | FREE | google_compute_region_ssl_policy |
| `google_compute_region_url_map` | FREE | google_compute_region_url_map |
| `google_compute_resource_policy` | FREE | google_compute_resource_policy |
| `google_compute_route` | FREE | GCP Compute Route |
| `google_compute_router` | FREE | GCP Compute Router |
| `google_compute_router_interface` | FREE | google_compute_router_interface |
| `google_compute_router_nat` | STATIC | GCP Cloud NAT |
| `google_compute_router_peer` | FREE | GCP Compute Router Peer |
| `google_compute_security_policy` | FREE | google_compute_security_policy |
| `google_compute_security_policy_advanced` | STATIC | GCP Cloud Armor Policy (Standard tier) |
| `google_compute_shared_vpc_host_project` | FREE | google_compute_shared_vpc_host_project |
| `google_compute_shared_vpc_service_project` | FREE | google_compute_shared_vpc_service_project |
| `google_compute_snapshot_iam_binding` | FREE | google_compute_snapshot_iam_binding |
| `google_compute_snapshot_iam_member` | FREE | google_compute_snapshot_iam_member |
| `google_compute_snapshot_iam_policy` | FREE | google_compute_snapshot_iam_policy |
| `google_compute_ssl_certificate` | FREE | google_compute_ssl_certificate |
| `google_compute_ssl_policy` | FREE | google_compute_ssl_policy |
| `google_compute_subnetwork` | FREE | GCP Compute Subnetwork |
| `google_compute_subnetwork_iam_binding` | FREE | google_compute_subnetwork_iam_binding |
| `google_compute_subnetwork_iam_member` | FREE | google_compute_subnetwork_iam_member |
| `google_compute_subnetwork_iam_policy` | FREE | google_compute_subnetwork_iam_policy |
| `google_compute_target_grpc_proxy` | FREE | google_compute_target_grpc_proxy |
| `google_compute_target_http_proxy` | FREE | GCP Compute Target Http Proxy |
| `google_compute_target_https_proxy` | FREE | GCP Compute Target Https Proxy |
| `google_compute_target_pool` | FREE | GCP Compute Target Pool |
| `google_compute_target_ssl_proxy` | FREE | GCP Compute Target Ssl Proxy |
| `google_compute_target_tcp_proxy` | FREE | GCP Compute Target Tcp Proxy |
| `google_compute_url_map` | FREE | GCP Compute Url Map |
| `google_compute_vpn_tunnel` | STATIC | GCP Cloud VPN Tunnel |
| `google_container_analysis_note` | FREE | google_container_analysis_note |
| `google_container_analysis_occurrence` | FREE | google_container_analysis_occurrence |
| `google_container_cluster` | LIVE | GCP GKE Cluster |
| `google_container_node_pool` | LIVE | GCP GKE Node Pool |
| `google_container_registry` | FREE | google_container_registry |
| `google_dataflow_flex_template_job` | FREE | google_dataflow_flex_template_job |
| `google_dataflow_job` | LIVE | GCP Dataflow Job |
| `google_dataproc_autoscaling_policy` | FREE | google_dataproc_autoscaling_policy |
| `google_dataproc_cluster` | STATIC | GCP Dataproc Cluster |
| `google_dataproc_cluster_iam_binding` | FREE | google_dataproc_cluster_iam_binding |
| `google_dataproc_cluster_iam_member` | FREE | google_dataproc_cluster_iam_member |
| `google_dataproc_cluster_iam_policy` | FREE | google_dataproc_cluster_iam_policy |
| `google_dataproc_job_iam_binding` | FREE | google_dataproc_job_iam_binding |
| `google_dataproc_job_iam_member` | FREE | google_dataproc_job_iam_member |
| `google_dataproc_job_iam_policy` | FREE | google_dataproc_job_iam_policy |
| `google_dataproc_metastore_federation` | FREE | google_dataproc_metastore_federation |
| `google_dataproc_metastore_service` | LIVE | GCP Dataproc Metastore Service |
| `google_datastore_index` | FREE | google_datastore_index |
| `google_dns_managed_zone` | LIVE | GCP Cloud DNS Managed Zone |
| `google_dns_policy` | FREE | GCP Dns Policy |
| `google_dns_record_set` | FREE | GCP Dns Record Set |
| `google_dns_response_policy` | FREE | google_dns_response_policy |
| `google_dns_response_policy_rule` | FREE | google_dns_response_policy_rule |
| `google_endpoints_service` | FREE | google_endpoints_service |
| `google_essential_contacts_contact` | FREE | google_essential_contacts_contact |
| `google_eventarc_channel` | FREE | google_eventarc_channel |
| `google_eventarc_google_channel_config` | FREE | google_eventarc_google_channel_config |
| `google_eventarc_trigger` | FREE | google_eventarc_trigger |
| `google_filestore_instance` | LIVE | GCP Filestore Instance |
| `google_firebase_android_app` | FREE | google_firebase_android_app |
| `google_firebase_apple_app` | FREE | google_firebase_apple_app |
| `google_firebase_project` | FREE | google_firebase_project |
| `google_firebase_web_app` | FREE | google_firebase_web_app |
| `google_firestore_database` | STATIC | GCP Firestore Database |
| `google_firestore_document` | FREE | google_firestore_document |
| `google_firestore_field` | FREE | google_firestore_field |
| `google_firestore_index` | FREE | google_firestore_index |
| `google_folder` | FREE | google_folder |
| `google_folder_iam_binding` | FREE | google_folder_iam_binding |
| `google_folder_iam_member` | FREE | google_folder_iam_member |
| `google_folder_iam_policy` | FREE | google_folder_iam_policy |
| `google_folder_organization_policy` | FREE | google_folder_organization_policy |
| `google_identity_platform_config` | FREE | google_identity_platform_config |
| `google_identity_platform_tenant` | FREE | google_identity_platform_tenant |
| `google_kms_autokey_config` | FREE | google_kms_autokey_config |
| `google_kms_crypto_key_iam_binding` | FREE | google_kms_crypto_key_iam_binding |
| `google_kms_crypto_key_iam_member` | FREE | GCP Kms Crypto Key Iam Member |
| `google_kms_crypto_key_iam_policy` | FREE | google_kms_crypto_key_iam_policy |
| `google_kms_crypto_key_version` | LIVE | GCP KMS Crypto Key Version |
| `google_kms_key_handle` | FREE | google_kms_key_handle |
| `google_kms_key_ring` | FREE | GCP Kms Key Ring |
| `google_kms_key_ring_iam_binding` | FREE | google_kms_key_ring_iam_binding |
| `google_kms_key_ring_iam_member` | FREE | google_kms_key_ring_iam_member |
| `google_kms_key_ring_iam_policy` | FREE | google_kms_key_ring_iam_policy |
| `google_kms_secret_ciphertext` | FREE | google_kms_secret_ciphertext |
| `google_logging_billing_account_exclusion` | FREE | google_logging_billing_account_exclusion |
| `google_logging_billing_account_sink` | FREE | google_logging_billing_account_sink |
| `google_logging_folder_exclusion` | FREE | google_logging_folder_exclusion |
| `google_logging_folder_sink` | FREE | google_logging_folder_sink |
| `google_logging_metric` | FREE | GCP Logging Metric |
| `google_logging_organization_exclusion` | FREE | google_logging_organization_exclusion |
| `google_logging_organization_sink` | FREE | google_logging_organization_sink |
| `google_logging_project_bucket` | STATIC | GCP Cloud Logging Bucket |
| `google_logging_project_exclusion` | FREE | google_logging_project_exclusion |
| `google_logging_project_sink` | FREE | GCP Logging Project Sink |
| `google_looker_instance` | LIVE | GCP Looker (Google Cloud core) Instance |
| `google_memcache_instance` | STATIC | GCP Memorystore Memcached |
| `google_monitoring_alert_policy` | FREE | google_monitoring_alert_policy |
| `google_monitoring_custom_service` | FREE | google_monitoring_custom_service |
| `google_monitoring_dashboard` | FREE | google_monitoring_dashboard |
| `google_monitoring_group` | FREE | google_monitoring_group |
| `google_monitoring_notification_channel` | FREE | google_monitoring_notification_channel |
| `google_monitoring_slo` | FREE | google_monitoring_slo |
| `google_monitoring_uptime_check_config` | FREE | google_monitoring_uptime_check_config |
| `google_notebooks_instance` | LIVE | GCP Vertex AI Workbench Instance |
| `google_org_policy_policy` | FREE | google_org_policy_policy |
| `google_organization_iam_binding` | FREE | google_organization_iam_binding |
| `google_organization_iam_custom_role` | FREE | google_organization_iam_custom_role |
| `google_organization_iam_member` | FREE | google_organization_iam_member |
| `google_organization_iam_policy` | FREE | google_organization_iam_policy |
| `google_organization_policy` | FREE | google_organization_policy |
| `google_privateca_certificate_authority` | LIVE | GCP Certificate Authority Service CA |
| `google_project` | FREE | google_project |
| `google_project_default_service_accounts` | FREE | google_project_default_service_accounts |
| `google_project_iam_audit_config` | FREE | google_project_iam_audit_config |
| `google_project_iam_binding` | FREE | GCP Project Iam Binding |
| `google_project_iam_custom_role` | FREE | google_project_iam_custom_role |
| `google_project_iam_member` | FREE | GCP Project Iam Member |
| `google_project_iam_policy` | FREE | GCP Project Iam Policy |
| `google_project_organization_policy` | FREE | google_project_organization_policy |
| `google_project_service` | FREE | google_project_service |
| `google_project_usage_export_bucket` | FREE | google_project_usage_export_bucket |
| `google_pubsub_schema` | FREE | google_pubsub_schema |
| `google_pubsub_subscription` | FREE | GCP Pubsub Subscription |
| `google_pubsub_subscription_iam_binding` | FREE | google_pubsub_subscription_iam_binding |
| `google_pubsub_subscription_iam_member` | FREE | google_pubsub_subscription_iam_member |
| `google_pubsub_subscription_iam_policy` | FREE | google_pubsub_subscription_iam_policy |
| `google_pubsub_topic` | LIVE | GCP Pub/Sub Topic |
| `google_pubsub_topic_iam_binding` | FREE | google_pubsub_topic_iam_binding |
| `google_pubsub_topic_iam_member` | FREE | GCP Pubsub Topic Iam Member |
| `google_pubsub_topic_iam_policy` | FREE | google_pubsub_topic_iam_policy |
| `google_recaptcha_enterprise_key` | FREE | google_recaptcha_enterprise_key |
| `google_redis_cluster` | LIVE | GCP Memorystore Redis Cluster |
| `google_redis_instance` | LIVE | GCP Memorystore for Redis |
| `google_secret_manager_secret` | LIVE | GCP Secret Manager Secret |
| `google_secret_manager_secret_iam_binding` | FREE | google_secret_manager_secret_iam_binding |
| `google_secret_manager_secret_iam_member` | FREE | google_secret_manager_secret_iam_member |
| `google_secret_manager_secret_iam_policy` | FREE | google_secret_manager_secret_iam_policy |
| `google_secret_manager_secret_version` | FREE | GCP Secret Manager Secret Version |
| `google_service_account` | FREE | GCP Service Account |
| `google_service_account_iam_binding` | FREE | google_service_account_iam_binding |
| `google_service_account_iam_member` | FREE | google_service_account_iam_member |
| `google_service_account_iam_policy` | FREE | google_service_account_iam_policy |
| `google_service_account_key` | FREE | google_service_account_key |
| `google_service_networking_connection` | FREE | google_service_networking_connection |
| `google_sourcerepo_repository` | FREE | google_sourcerepo_repository |
| `google_spanner_database` | FREE | google_spanner_database |
| `google_spanner_database_iam_binding` | FREE | google_spanner_database_iam_binding |
| `google_spanner_database_iam_member` | FREE | google_spanner_database_iam_member |
| `google_spanner_database_iam_policy` | FREE | google_spanner_database_iam_policy |
| `google_spanner_instance` | LIVE | GCP Spanner Instance |
| `google_spanner_instance_iam_binding` | FREE | google_spanner_instance_iam_binding |
| `google_spanner_instance_iam_member` | FREE | google_spanner_instance_iam_member |
| `google_spanner_instance_iam_policy` | FREE | google_spanner_instance_iam_policy |
| `google_sql_database_instance` | LIVE | GCP Cloud SQL Instance |
| `google_storage_bucket` | LIVE | GCP Cloud Storage Bucket |
| `google_storage_bucket_acl` | FREE | google_storage_bucket_acl |
| `google_storage_bucket_iam_binding` | FREE | GCP Storage Bucket Iam Binding |
| `google_storage_bucket_iam_member` | FREE | GCP Storage Bucket Iam Member |
| `google_storage_bucket_iam_policy` | FREE | GCP Storage Bucket Iam Policy |
| `google_storage_bucket_object` | FREE | google_storage_bucket_object |
| `google_storage_default_object_access_control` | FREE | google_storage_default_object_access_control |
| `google_storage_default_object_acl` | FREE | GCP Storage Default Object Acl |
| `google_storage_hmac_key` | FREE | google_storage_hmac_key |
| `google_storage_insights_report_config` | FREE | google_storage_insights_report_config |
| `google_storage_notification` | FREE | google_storage_notification |
| `google_storage_object_access_control` | FREE | google_storage_object_access_control |
| `google_storage_object_acl` | FREE | google_storage_object_acl |
| `google_storage_transfer_job` | FREE | google_storage_transfer_job |
| `google_tags_tag_binding` | FREE | google_tags_tag_binding |
| `google_tags_tag_key` | FREE | google_tags_tag_key |
| `google_tags_tag_value` | FREE | google_tags_tag_value |
| `google_vertex_ai_endpoint` | LIVE | GCP Vertex AI Endpoint |
| `google_vmwareengine_cluster` | STATIC | GCP VMware Engine Cluster |
| `google_vpc_access_connector` | FREE | google_vpc_access_connector |
| `google_workbench_instance` | LIVE | GCP Vertex AI Workbench Instance (v2) |
| `google_workflows_workflow` | STATIC | GCP Workflows Workflow |

**GCP totals:** 29 LIVE · 21 STATIC · 256 FREE

---

**Grand total: 1340 resources.** 176 LIVE · 87 STATIC · 1077 FREE.
