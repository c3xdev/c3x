# C3X Example — Terragrunt (multi-environment)
#
# Run: c3x estimate --path .

locals {
  aws_region = "us-east-1"
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
provider "aws" {
  region                      = "${local.aws_region}"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}
EOF
}
