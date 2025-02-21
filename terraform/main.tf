terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.54.1"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = var.aws_region
}

resource "aws_key_pair" "ssh_key" {
  key_name = "ssh_key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_security_group" "ssh" {
  name = "allow-ssh"

  ingress {
    cidr_blocks = ["0.0.0.0/0"]

    from_port = 22
    to_port   = 22
    protocol  = "tcp"
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "lb" {
  name = "allow-lb"

  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port = 9020
    to_port   = 9020
    protocol  = "tcp"
  }

  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port = 80
    to_port   = 80
    protocol  = "tcp"
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "worker" {
  name = "allow-worker"

  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port = 5000
    to_port   = 5000 + var.n_workers - 1
    protocol  = "tcp"
  }

  ingress {
    cidr_blocks = ["0.0.0.0/0"]
    from_port = 80
    to_port   = 80
    protocol  = "tcp"
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "scheduler" {
  ami           = var.ami_id
  instance_type = var.lb_instance_type
  key_name      = aws_key_pair.ssh_key.key_name
  security_groups = [
    aws_security_group.ssh.name,
    aws_security_group.lb.name
  ]
  instance_market_options {
    market_type = "spot"
  }

  tags = {
    Name = "scheduler"
  }
}

resource "aws_instance" "workers" {
  count         = var.n_workers
  ami           = var.ami_id
  instance_type = var.worker_instance_type
  key_name      = aws_key_pair.ssh_key.key_name
  security_groups = [
    aws_security_group.ssh.name,
    aws_security_group.worker.name
  ]
  instance_market_options {
    market_type = "spot"
  }

  root_block_device {
    volume_size = 150
  }

  tags = {
    Name = "worker-${count.index + 1}"
  }
}
