# C3X Example — SaaS Application Stack on AWS
#
# Try these commands:
#   c3x estimate --path .
#   c3x estimate --path . --usage-file c3x-usage.yml
#   c3x estimate --path . --recommend
#   c3x estimate --path . --budget 800
#   c3x estimate --path . --what-if 'aws_instance.api.instance_type=m7i.xlarge'

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

# --- API Server ---

resource "aws_instance" "api" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "m5.xlarge"

  root_block_device {
    volume_size = 100
    volume_type = "gp2"
  }

  ebs_block_device {
    device_name = "/dev/sdf"
    volume_type = "gp3"
    volume_size = 500
  }

  tags = {
    Name        = "api-server"
    Environment = "production"
  }
}

# --- Database ---

resource "aws_db_instance" "postgres" {
  identifier          = "app-database"
  engine              = "postgres"
  engine_version      = "16"
  instance_class      = "db.r5.large"
  allocated_storage   = 200
  storage_type        = "gp3"
  multi_az            = true
  skip_final_snapshot = true

  tags = {
    Name = "app-database"
  }
}

# --- Networking ---

resource "aws_nat_gateway" "main" {
  allocation_id = "eipalloc-12345678"
  subnet_id     = "subnet-12345678"

  tags = {
    Name = "main-nat"
  }
}

resource "aws_lb" "api" {
  name               = "api-lb"
  internal           = false
  load_balancer_type = "application"
  subnets            = ["subnet-12345678", "subnet-87654321"]

  tags = {
    Name = "api-lb"
  }
}

# --- Storage ---

resource "aws_s3_bucket" "data" {
  bucket = "company-data-store"

  tags = {
    Name = "data-store"
  }
}

# --- Monitoring ---

resource "aws_cloudwatch_log_group" "app" {
  name              = "/app/api"
  retention_in_days = 30
}
