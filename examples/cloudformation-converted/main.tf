# C3X Example — CloudFormation-to-Terraform converted resources
#
# Run: c3x estimate --path .

provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

resource "aws_dynamodb_table" "events" {
  name         = "app-events"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "eventId"
  range_key    = "timestamp"

  attribute {
    name = "eventId"
    type = "S"
  }
  attribute {
    name = "timestamp"
    type = "N"
  }

  point_in_time_recovery {
    enabled = true
  }
}

resource "aws_dynamodb_table" "sessions" {
  name         = "app-sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "sessionId"

  attribute {
    name = "sessionId"
    type = "S"
  }
}

resource "aws_dynamodb_table" "analytics" {
  name           = "app-analytics"
  billing_mode   = "PROVISIONED"
  read_capacity  = 25
  write_capacity = 50
  hash_key       = "metricId"
  range_key      = "period"

  attribute {
    name = "metricId"
    type = "S"
  }
  attribute {
    name = "period"
    type = "S"
  }
}
