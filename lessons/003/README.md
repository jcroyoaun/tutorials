# Kubernetes QoS Classes Demonstration

A chaos engineering resource consumer application that demonstrates Kubernetes Quality of Service (QoS) classes and pod eviction behavior under memory pressure.

## Cloud Native LATAM Summit 2025

This project was created for the [Cloud Native LATAM Summit 2025](https://community.cncf.io/events/details/cncf-cloud-native-latam-presents-cloud-native-latam-summit/).

**Session**: [Debugging Kubernetes in Production: Why Your Pod Dies Even Though Nodes Have Sufficient Memory](https://sessionize.com/s/juan-carlos-martinez-carrillo/debugging-kubernetes-in-production-why-your-pod-di/153441)

**Additional Tool**: [K8s Describe Nodes Parser](https://k8sdescribenodes.jcroyoaun.com/) - Tool for analyzing node output and identifying resource pressure issues.

## QoS Class Deployments

This project deploys the same Go application with different resource configurations:

### 1. Guaranteed (requests == limits)
- **Endpoint**: `https://guaranteed.resourcehog.jcroyoaun.com`
- **Resources**: 1024Mi memory, 200m CPU (hard limits)
- **Eviction Priority**: Last to evict

### 2. Burstable (requests < limits)
- **Endpoint**: `https://burstable.resourcehog.jcroyoaun.com`
- **Resources**: 128Mi-512Mi memory, 100m-500m CPU
- **Eviction Priority**: Middle priority

### 3. BestEffort (no requests/limits)
- **Endpoint**: `https://besteffort.resourcehog.jcroyoaun.com`
- **Resources**: Unbounded
- **Eviction Priority**: First to evict

### 4. Troublemaker (massive overcommit)
- **Endpoint**: `https://chaos.resourcehog.jcroyoaun.com`
- **Resources**: 345Mi request, 5Gi limit (62x overcommit)
- **QoS Class**: Burstable

## Live Demo Scenario

All pods are scheduled on a **c6a.large** node in AWS with **3.1GiB allocatable memory**.

### Step 1: Consume most of the node's memory
```bash
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/memory -d '{"mb": 2400}'
```
This allocates 2400 MiB out of the 3.1GiB available.

### Step 2: Consume the remaining memory
```bash
curl -X POST https://guaranteed.resourcehog.jcroyoaun.com/api/memory -d '{"mb": 750}'
```
This consumes the rest of the allocatable memory.

### What Happens Next

**The troublemaker pod (Burstable QoS) will crash due to insufficient memory.**

### Why Does This Happen?

1. **Node memory pressure**: The node runs out of memory even though individual pods haven't exceeded their limits
2. **Grafana won't show the spike**: Node-level metrics show available memory, but don't reveal memory pressure at the kernel level
3. **Eviction order**: Kubernetes evicts Burstable pods that exceed their requests before touching Guaranteed pods
4. **The troublemaker is vulnerable**: It requested only 345Mi but tried to use 2400Mi, making it a prime eviction candidate

The guaranteed pod stays alive because its request equals its limit (1024Mi), and it's within that boundary. The troublemaker pod gets evicted because it's a Burstable pod consuming far more than its request during memory pressure.

## Quick Start

### Deploy Infrastructure
```bash
cd infra
terraform init
terraform apply
```

### Deploy Applications
```bash
cd k8s
kubectl apply -f .
```

### Verify QoS Classes
```bash
kubectl get pods -n qos-demo -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass
```

### Monitor Pod Status
```bash
kubectl get pods -n qos-demo -w
kubectl get events -n qos-demo --sort-by='.lastTimestamp'
```

## Key Takeaways

1. **QoS Classes determine eviction order**: Guaranteed → Burstable → BestEffort
2. **Memory pressure is invisible to standard metrics**: Grafana and Prometheus won't show the real issue
3. **Overcommitted nodes are dangerous**: Total pod limits can exceed node capacity
4. **Requests are promises, limits are boundaries**: Pods exceeding requests during pressure get evicted first

## Cleanup

```bash
kubectl delete namespace qos-demo
cd infra && terraform destroy
```

