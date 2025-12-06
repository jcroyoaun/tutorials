apiVersion: karpenter.k8s.aws/v1
kind: EC2NodeClass
metadata:
  name: general-amd64 
spec:
  amiSelectorTerms:
    - alias: al2023@latest
  amiFamily: AL2023
  role: ${karpenter_role}
  subnetSelectorTerms:
    - tags:
        karpenter.sh/discovery: ${discovery_tag}
  securityGroupSelectorTerms:
    - tags:
        karpenter.sh/discovery: ${discovery_tag}
  blockDeviceMappings:
    - deviceName: /dev/xvda
      ebs:
        volumeSize: 50Gi
        volumeType: gp3
        deleteOnTermination: true
