# C3X Example — Terragrunt prod environment

include {
  path = find_in_parent_folders()
}

terraform {
  source = "${get_parent_terragrunt_dir()}/modules/app"
}

inputs = {
  environment       = "prod"
  instance_type     = "m5.xlarge"
  db_instance_class = "db.r5.large"
}
