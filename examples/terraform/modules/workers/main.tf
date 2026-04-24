# Example Terraform module — creates multiple instances using for_each

locals {
  environments = ["staging", "canary", "production"]
}

resource "aws_instance" "worker" {
  for_each = toset(local.environments)

  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.medium"

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  tags = {
    Name        = "worker-${each.key}"
    Environment = each.key
  }
}
