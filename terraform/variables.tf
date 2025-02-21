variable "aws_region" {
  description = "The AWS region to deploy in"
  default     = "eu-central-1"
}

variable "lb_instance_type" {
  description = "The instance type to use for the EC2 instance of the scheduler"
  default     = "m5.xlarge"
}

variable "worker_instance_type" {
  description = "The instance type to use for the EC2 instances of the workers"
  default     = "m5.xlarge"
}

variable "ami_id" {
  description = "The AMI ID to use for the EC2 instances"
  default     = "ami-00cf59bc9978eb266"
}

variable "n_workers" {
  description = "The number of worker instances to deploy"
  default     = 5
}
