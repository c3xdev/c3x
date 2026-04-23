provider "aws" {
  region                      = "us-east-1" # <<<<< Try changing this to eu-west-1 to compare the costs
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"

  default_tags {
    tags = {
      DefaultNotOverride = "defaultnotoverride"
      DefaultOverride    = "defaultoverride"
    }
  }
}

resource "aws_instance" "app_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "m5.4xlarge"
  volume_tags = {
    "baz" = "bat"
  }

  root_block_device {
    volume_size = 51
  }

  ebs_block_device {
    device_name = "data_vol"
    volume_type = "gp2"
    volume_size = 1000
    iops        = 800
  }

  tags = {
    "foo" = "bar"
  }
}
