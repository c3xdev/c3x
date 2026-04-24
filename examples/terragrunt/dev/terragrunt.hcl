# C3X Example — Terragrunt dev environment

include {
  path = find_in_parent_folders()
}

terraform {
  source = "${get_parent_terragrunt_dir()}/modules/app"
}

inputs = {
  environment       = "dev"
  instance_type     = "t3.small"
  db_instance_class = "db.t3.micro"
}
