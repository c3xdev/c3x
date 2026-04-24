provider "aws" {
  region                      = "us-east-1" # <<<<< Try changing this to eu-west-1 to compare the costs
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

module "small_app" {
  source              = "./modules/instance"
  instance_type       = "m5.4xlarge"
  volume_type         = "io1"
  child_instance_type = "m5.8xlarge"
}

module "small_app_gp2" {
  source              = "./modules/instance"
  instance_type       = "m5.4xlarge"
  volume_type         = "gp2"
  child_instance_type = "m5.8xlarge"
}

module "big_app" {
  source              = "./modules/instance"
  instance_type       = "m5.8xlarge"
  volume_type         = "gp2"
  child_instance_type = "m5.4xlarge"
}

module "big_app_gp2" {
  source              = "./modules/instance"
  instance_type       = "m5.8xlarge"
  volume_type         = "gp2"
  child_instance_type = "m5.4xlarge"
}

module "big_app_with_output" {
  source              = "./modules/instance"
  instance_type       = module.big_app_gp2.parent_instance_type
  volume_type         = "gp2"
  child_instance_type = "m5.4xlarge"
}

resource "aws_lambda_function" "event_handler" {
  function_name = "event_handler"
  role          = "arn:aws:lambda:us-east-1:aws:resource-id"
  handler       = "index.handler"
  runtime       = "nodejs20.x"
  filename      = "function.zip"
  memory_size   = var.memory_size
}

resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = var.instance_type

  root_block_device {
    volume_size = 50
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = var.volume_type # <<<<< Try changing this to gp2 to compare costs
    volume_size = 1000
    iops        = 800
  }
}

variable "memory_size" {
  default = 1024
}

variable "instance_type" {
  default = "m5.2xlarge"
}

variable "volume_type" {
  default = "io1"
}
