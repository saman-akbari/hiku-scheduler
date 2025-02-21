output "scheduler_url" {
  value = aws_instance.scheduler.public_dns
}

output "worker_url" {
  value = [for instance in aws_instance.workers : instance.public_dns]
}
