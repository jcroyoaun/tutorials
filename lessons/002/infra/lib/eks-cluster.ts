import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as eks from '@aws-cdk/aws-eks-v2-alpha';
import { KubectlV32Layer } from '@aws-cdk/lambda-layer-kubectl-v32';
import { Size } from 'aws-cdk-lib';
import { Construct } from 'constructs';

export class EksV2ClusterStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // 1. Create a VPC with private and public subnets across 3 AZs
    const vpc = new ec2.Vpc(this, 'EksVpc', {
      maxAzs: 3,
      natGateways: 1,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: 'public',
          subnetType: ec2.SubnetType.PUBLIC,
        },
        {
          cidrMask: 20,
          name: 'private',
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
        }
      ]
    });

    // Create cluster admin role
    const clusterAdminRole = new iam.Role(this, 'ClusterAdminRole', {
      assumedBy: new iam.AccountRootPrincipal(),
      description: 'EKS cluster admin role'
    });

    // Create cluster role with required EKS policies
    const clusterRole = new iam.Role(this, 'ClusterRole', {
      assumedBy: new iam.ServicePrincipal('eks.amazonaws.com'),
      managedPolicies: [
        // Regular cluster policies
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEKSClusterPolicy'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEKSVPCResourceController')
      ]
    });

    // Create node group role with all required policies
    const nodeRole = new iam.Role(this, 'NodeGroupRole', {
      assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com'),
      managedPolicies: [
        // Node group policies
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEKSWorkerNodePolicy'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEKS_CNI_Policy'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonEC2ContainerRegistryReadOnly'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMManagedInstanceCore')
      ]
    });

    // 2. Create the EKS cluster using aws-eks-v2-alpha module
    const cluster = new eks.Cluster(this, 'EksCluster', {
      version: eks.KubernetesVersion.V1_32,
      vpc,
      clusterName: 'kcd-gdl-demo',
      vpcSubnets: [{ subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS }],
      defaultCapacityType: eks.DefaultCapacityType.NODEGROUP,
      defaultCapacity: 0, // We'll create the node group ourselves
      role: clusterRole,
      mastersRole: clusterAdminRole,
      kubectlProviderOptions: {
        kubectlLayer: new KubectlV32Layer(this, 'KubectlLayer'),
        memory: Size.gibibytes(2),
      },
      // AWS ALB Controller
      albController: {
        version: eks.AlbControllerVersion.V2_8_2,
      }
    });

    // Create node group with the custom role
    const mainNodeGroup = cluster.addNodegroupCapacity('MainNodeGroup', {
      nodeRole: nodeRole,
      instanceTypes: [ec2.InstanceType.of(ec2.InstanceClass.M5, ec2.InstanceSize.LARGE)],
      minSize: 2,
      maxSize: 4,
      desiredSize: 2,
      diskSize: 100,
      amiType: eks.NodegroupAmiType.AL2_X86_64,
      enableNodeAutoRepair: true,
    });

    // Add your specific IAM identity as cluster admin
    cluster.grantAccess('YourUserAccess', 'arn:aws:iam::443311183770:user/iamadmin', [
      eks.AccessPolicy.fromAccessPolicyName('AmazonEKSClusterAdminPolicy', {
        accessScopeType: eks.AccessScopeType.CLUSTER,
      }),
    ]);
    
    // EBS CSI Driver setup with OIDC - using correct policy path
    const ebsCsiServiceAccount = cluster.addServiceAccount('EbsCsiDriverSA', {
      name: 'ebs-csi-controller-sa',
      namespace: 'kube-system',
    });

    // Using the CORRECT policy ARN with service-role path
    ebsCsiServiceAccount.role.addManagedPolicy(
      iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AmazonEBSCSIDriverPolicy')
    );

    const ebsCsiDriver = cluster.addHelmChart('AwsEbsCsiDriver', {
      chart: 'aws-ebs-csi-driver',
      repository: 'https://kubernetes-sigs.github.io/aws-ebs-csi-driver',
      namespace: 'kube-system',
      values: {
        controller: {
          serviceAccount: {
            create: false,
            name: ebsCsiServiceAccount.serviceAccountName
          }
        },
        storageClasses: [
          {
            name: 'gp3',
            annotations: {
              'storageclass.kubernetes.io/is-default-class': 'true'
            },
            volumeBindingMode: 'WaitForFirstConsumer',
            reclaimPolicy: 'Delete',
            parameters: {
              type: 'gp3',
              allowVolumeExpansion: 'true'
            }
          }
        ]
      }
    });

    // Make sure EBS CSI Driver is installed after the node group
    ebsCsiDriver.node.addDependency(mainNodeGroup);

    // Install CoreDNS addon
    const coreDnsAddon = new eks.Addon(this, 'CoreDNSAddon', {
      cluster,
      addonName: 'coredns',
    });

    // Configure VPC-CNI addon with settings compatible with Cilium
    const vpcCniAddon = new eks.Addon(this, 'VpcCniAddon', {
      cluster,
      addonName: 'vpc-cni',
    });

    // Install Cilium with Hubble for network policies
    const cilium = cluster.addHelmChart('Cilium', {
      chart: 'cilium',
      repository: 'https://helm.cilium.io/',
      namespace: 'kube-system',
      values: {
        cni: {
          chainingMode: 'aws-cni',
          exclusive: false
        },
        enableIPv4Masquerade: false,
        routingMode: 'native',
        // Enable Hubble for observability and network policy visualization
        hubble: {
          enabled: true,
          relay: {
            enabled: true
          },
          ui: {
            enabled: true
          }
        },
        // Enable network policies
        policyEnforcement: 'default'
      }
    });

    // Make sure Cilium is installed after VPC-CNI is configured
    cilium.node.addDependency(vpcCniAddon);

    // Update kube-proxy addon
    const kubeProxyAddon = new eks.Addon(this, 'KubeProxyAddon', {
      cluster,
      addonName: 'kube-proxy',
    });

    // Enable EKS Pod Identity
    const podIdentityAgent = new eks.Addon(this, 'PodIdentityAgent', {
      cluster,
      addonName: 'eks-pod-identity-agent',
    });

    // Install Metrics Server
    const metricsServer = cluster.addHelmChart('MetricsServer', {
      chart: 'metrics-server',
      repository: 'https://kubernetes-sigs.github.io/metrics-server',
      namespace: 'kube-system',
    });

    // Make sure Metrics Server is installed after the node group
    metricsServer.node.addDependency(mainNodeGroup);

    // Outputs for easy access
    new cdk.CfnOutput(this, 'ClusterName', {
      value: cluster.clusterName,
    });

    new cdk.CfnOutput(this, 'KubectlCommand', {
      value: `aws eks update-kubeconfig --name ${cluster.clusterName} --region ${this.region} --role-arn ${clusterAdminRole.roleArn}`,
    });

    new cdk.CfnOutput(this, 'ClusterAdminRoleArn', {
      value: clusterAdminRole.roleArn,
    });

    new cdk.CfnOutput(this, 'ClusterOidcIssuer', {
      value: cluster.clusterOpenIdConnectIssuerUrl,
    });
  }
}
