provider "aws" {
  region                      = "us-east-1" # <<<<< Try changing this to eu-west-1 to compare the costs
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "m5.4xlarge" # <<<<< Try changing this to m5.8xlarge to compare the costs

  root_block_device {
    volume_size = 50
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = "io1" # <<<<< Try changing this to gp2 to compare costs
    volume_size = 1000
    iops        = 800
  }
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  filename      = "function.zip"
  memory_size   = 1024 # <<<<< Try changing this to 512 to compare costs
}

output "aws_instance_type" {
  value = aws_instance.app_server.instance_type
}
