resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "m5.4xlarge"

  root_block_device {
    volume_size = 50
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = "io1"
    volume_size = 1000
    iops        = 800
  }
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  memory_size   = 1024
}

output "aws_instance_type" {
  value = aws_instance.app_server.instance_type
}
