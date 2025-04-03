# DEMO - Installing Cilium and Hubble on EKS

This demo was presented on Kubernetes Community Day in GDL Mexico (2025 edition), and its purpose is to explain how to use Cilium on an EKS cluster in combination with the AWS VPC CNI plugin.

## Pre-requisites:
* Cilium CLI - The installation guide for Cilium CLI can be found [here](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#install-the-cilium-cli)

* Hubble CLI - The Installation guide for Hubble CLI can be found [here](https://docs.cilium.io/en/stable/observability/hubble/setup/#install-the-hubble-client)


* npm and cdk - The EKS cluster in this demo is installed by CDK (along with Cilium installation). My blog has a tutorial on how to install npm in what I consider the "correct" way: [install npm](https://medium.com/@jcroyoaun/my-quick-typescript-setup-guide-748437193d64)
* * After npm has been installed, cdk installation is needed. To install cdk simply do
```bash
npm -g cdk
```

* Helm and Kubectl CLIs - This demo contains a frontend, backend and database applications to test Hubble's observability capabilities. I'll assume you already have these installed on your machine, but using `brew` or `apt` would suffice.

 
## Installation Steps:

### Install Infra
1. To deploy a demo cluster go to:
```bash
cd infra
```

2. Set your AWS acconut and region defaults:
```bash
export CDK_DEFAULT_REGION="us-east-1"
export CDK_DEFAULT_ACCOUNT="<youraccountnumber>" #12345678
```

3. Make sure your preferred AWS authentication method is set:
```bash
aws sts get-caller-identity
{
    "UserId": "AIDAWON3H26NCGWJLD577",
    "Account": "123456789",
    "Arn": "arn:aws:iam::123456789:user/myuser"
}
``` 

4. Then run the following commands

```bash
npm ci
cdk deploy
```

## Verifying Cilium was deployed
The CDK code contains a Cilium installation that works in EKS, as it chains Cilium's CNI with AWS VPC CNI in a Hybrid mode:
* https://docs.cilium.io/en/stable/installation/cni-chaining-aws-cni/

To verify Cilium was deployed:
```bash
❯ k get pods -n kube-system | grep cilium
cilium-envoy-ncj76                                                1/1     Running   0          3h48m
cilium-envoy-tj2bg                                                1/1     Running   0          3h48m
cilium-nz82d                                                      1/1     Running   0          3h48m
cilium-operator-59944f4b8f-jqlh4                                  1/1     Running   0          3h48m
cilium-operator-59944f4b8f-tnjtx                                  1/1     Running   0          3h48m
cilium-qgbt4                                                      1/1     Running   0          3h48m
```

NOTE: the configuration set in CDK is important, as Cilium working in exclusive mode, may cause unwanted behaviors when setting up the virtual network devices as well as for IP address management some basic networking functionalities in EKS.

Code snippet:
```typescript
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

```


## Install Applications
We want to install a frontend, backend and database on the cluster once its deployed, and after we've verified that Cilium is installed.

1. First install the database:

```bash
helm upgrade --install postgres ./helm/charts/microservice \
  -f ./helm/environments/local.yaml \
  -f ./helm/values/postgres.yaml \
  --namespace postgres --create-namespace
```


2. Then install the backend
```bash
helm upgrade --install exerciselib ./helm/charts/microservice \
  -f ./helm/environments/local.yaml \
  -f ./helm/values/exerciselib.yaml \
  --set service.container.image=jcroyoaun/exerciselib \
  --set image.tag=latest \
  --namespace exerciselib --create-namespace
```

3. Last, install the frontend.
```bash
helm upgrade --install frontend ./helm/charts/microservice \
  -f ./helm/environments/local.yaml \
  -f ./helm/values/frontend.yaml \
  --set service.container.image=jcroyoaun/frontend \
  --set image.tag=latest \
  --namespace frontend --create-namespace
```


## To test the installation
Once the pods from the 3 deployments are in running state, check the ingress and go to your browser:

```
❯ kubectl get ingress -n frontend
NAME               CLASS    HOSTS   ADDRESS                                                                  PORTS   AGE
frontend-ingress   <none>   *       k8s-frontend-frontend-ae4a928a81-255308396.us-east-1.elb.amazonaws.com   80      3h35m
```


## Observability with Hubble

### Option 1 - Checking logs using Hubble CLI
```
kubectl port-forward -n kube-system svc/hubble-relay 4245:80 &
hubble observe -f
```

### Option 2 - Using Hubble UI
```
cilium hubble ui &
```

Then navigate to localhost:12000

