# VPC Outputs
output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnet_ids" {
  value = module.vpc.private_subnet_ids
}

output "isolated_subnet_ids" {
  value = module.vpc.isolated_subnet_ids
}

# EKS Outputs
output "cluster_name" {
  value = module.eks.cluster_name
}

output "cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "kubectl_config_command" {
  description = "Run this command to configure kubectl"
  value       = module.eks.kubectl_config_command
}

# DNS Outputs
output "domain_name" {
  value = module.dns.domain_name
}

output "certificate_arn" {
  value = module.dns.certificate_arn
}

# ECR Outputs
output "ecr_repository_urls" {
  value = module.ecr.repository_urls
}

output "ecr_docker_login_command" {
  description = "Run this command to login to ECR"
  value       = module.ecr.docker_login_command
}
