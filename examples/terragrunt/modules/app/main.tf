variable "instance_type" {
  description = "EC2 instance type"
  type        = string
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

resource "aws_instance" "app" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = var.instance_type

  root_block_device {
    volume_size = 50
    volume_type = "gp3"
  }

  tags = {
    Name        = "${var.environment}-app"
    Environment = var.environment
  }
}

resource "aws_db_instance" "database" {
  identifier          = "${var.environment}-db"
  engine              = "postgres"
  engine_version      = "16"
  instance_class      = var.db_instance_class
  allocated_storage   = 50
  storage_type        = "gp3"
  skip_final_snapshot = true

  tags = {
    Name        = "${var.environment}-db"
    Environment = var.environment
  }
}
