provider "aws" {
  region                      = "us-east-1" # <<<<< Try changing this to eu-west-1 to compare the costs
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

data "local_file" "config" {
  filename = "../config/config.json"
}

locals {
  config = jsondecode(data.local_file.config.content)
}

resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = local.config.instance_type

  root_block_device {
    volume_size = 70
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = "io1"
    volume_size = 3000
    iops        = 1200 # <<<<< Try changing this to 10000 to compare costs
  }
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:account-id:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  memory_size   = 1024 # <<<<< Try changing this to 512 to compare costs
}
