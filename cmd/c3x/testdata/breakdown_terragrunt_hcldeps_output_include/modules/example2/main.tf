variable "instance_type" {
  description = "The EC2 instance type for the web app"
  type        = string
}

variable "root_block_device_volume_size" {
  description = "The size of the root block device volume for the web app EC2 instance"
  type        = number
}

variable "block_device_volume_size" {
  description = "The size of the block device volume for the web app EC2 instance"
  type        = number
}

variable "block_device_iops" {
  description = "The number of IOPS for the block device for the web app EC2 instance"
  type        = number
}

variable "event_handler_function_memory_size" {
  description = "The memory to allocate to the hello world Lambda function"
  type        = number
}

resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = var.instance_type

  root_block_device {
    volume_size = var.root_block_device_volume_size
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = "io1"
    volume_size = var.block_device_volume_size
    iops        = var.block_device_iops
  }
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  memory_size   = var.event_handler_function_memory_size
}

output "block_iops" {
  value = var.block_device_iops
}

output "test_object_type" {
  value = aws_instance.app_server
}
