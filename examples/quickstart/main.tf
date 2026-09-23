# A minimal project to try c3x against:
#
#   c3x estimate --path examples/quickstart
#
# Nothing here is applied or reachable. c3x reads the configuration
# statically, so no credentials, no terraform init, no state.

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "m5.xlarge"

  root_block_device {
    volume_type = "gp3"
    volume_size = 50
  }
}

resource "aws_db_instance" "primary" {
  engine            = "postgres"
  instance_class    = "db.t3.medium"
  allocated_storage = 50
}
