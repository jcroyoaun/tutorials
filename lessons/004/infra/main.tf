data "aws_caller_identity" "current" {}

# DNS Module
module "dns" {
  source = "git::https://github.com/jcroyoaun/terraform-aws-modules.git//modules/dns?ref=v1.0.19"

  subdomain             = local.dns.subdomain
  parent_domain         = local.dns.parent_domain
  parent_hosted_zone_id = local.dns.parent_hosted_zone_id
  env                   = local.env
}

# ECR Module
module "ecr" {
  source = "git::https://github.com/jcroyoaun/terraform-aws-modules.git//modules/ecr?ref=v1.0.19"

  env          = local.env
  repositories = local.ecr.repositories
}

# VPC Module
module "vpc" {
  source = "git::https://github.com/jcroyoaun/terraform-aws-modules.git//modules/vpc?ref=v1.0.19"

  region                  = local.region
  vpc_cidr                = local.vpc.cidr
  env                     = local.env
  azs                     = local.vpc.azs
  public_subnets          = local.vpc.public_subnets
  private_subnets         = local.vpc.private_subnets
  create_isolated_subnets = local.vpc.create_isolated_subnets
  isolated_subnet_cidrs   = local.vpc.isolated_subnet_cidrs
  cluster_name            = local.eks.cluster_name
  private_subnet_tags = {
    "karpenter.sh/discovery" = local.eks.cluster_name
  }
}

module "eks" {
  source = "git::https://github.com/jcroyoaun/terraform-aws-modules.git//modules/eks?ref=v1.0.21"

  region              = local.region
  env                 = local.env
  cluster_name        = local.eks.cluster_name
  vpc_id              = module.vpc.vpc_id
  private_subnet_ids  = module.vpc.private_subnet_ids
  public_subnet_ids   = module.vpc.public_subnet_ids
  kubernetes_version  = local.eks.kubernetes_version
  node_group_name     = local.eks.node_group.name
  node_instance_types = local.eks.node_group.instance_types
  node_capacity_type  = local.eks.node_group.capacity_type
  node_scaling_config = local.eks.node_group.scaling_config
  node_disk_size      = local.eks.node_group.disk_size
  node_taints         = local.eks.node_group.taints
  node_labels         = local.eks.node_group.labels

  cluster_admin_arns = local.eks.cluster_admin_arns
  addon_versions     = local.eks.addon_versions

  charts = local.eks.charts

  pod_identity_associations = local.eks.pod_identity_associations

  external_dns_domain_filters   = [module.dns.external_dns_domain_filter]
  external_dns_hosted_zone_arns = [module.dns.external_dns_hosted_zone_arn]

  # Add tolerations and node selector to CoreDNS so it runs on system nodes
  coredns_tolerations   = local.common_tolerations
  coredns_node_selector = local.common_node_selector
}

module "karpenter_manifests" {
  source = "git::https://github.com/jcroyoaun/terraform-aws-modules.git//modules/k8s-manifests?ref=v1.0.20" 

  cluster_name           = module.eks.cluster_name
  cluster_region         = local.region
  cluster_endpoint       = module.eks.cluster_endpoint
  cluster_ca_certificate = module.eks.cluster_certificate_authority_data

  manifests = {
    karpenter_ec2nodeclass = {
      file_path = "${path.module}/manifests/karpenter-ec2nodeclass.yaml.tpl"
      vars = {
        karpenter_role = module.eks.karpenter_node_role_name
        discovery_tag  = local.eks.cluster_name
      }
    }
    karpenter_nodepool = {
      file_path = "${path.module}/manifests/karpenter-nodepool.yaml.tpl"
      vars = {
        # Add any vars if needed in the future
      }
    }
  }

  cluster_ready_dependency = module.eks.karpenter_helm_release
}
