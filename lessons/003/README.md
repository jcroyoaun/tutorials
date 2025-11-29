# Kubernetes QoS Classes Demonstration

A chaos engineering resource consumer application that demonstrates Kubernetes Quality of Service (QoS) classes through controlled resource consumption patterns. This project deploys the same Go application with different resource configurations to showcase how Kubernetes handles pod eviction priorities.

## Cloud Native LATAM Summit 2025

This project was created for the [Cloud Native LATAM Summit 2025](https://community.cncf.io/events/details/cncf-cloud-native-latam-presents-cloud-native-latam-summit/), a virtual three-day conference focused on cloud native practices, tools, and culture in Latin America.

**Session**: [Debugging Kubernetes in Production: Why Your Pod Dies Even Though Nodes Have Sufficient Memory](https://sessionize.com/s/juan-carlos-martinez-carrillo/debugging-kubernetes-in-production-why-your-pod-di/153441)

This session delivers recommendations for deploying reliable services in Kubernetes, covering QoS classes, overcommitted nodes, and memory pressure. Through a real production incident walkthrough, you'll learn how to efficiently set requests and limits, use node anti-affinity rules, and monitor your clusters effectively.

**Additional Tool**: To help debug these issues, check out the [K8s Describe Nodes Parser](https://k8sdescribenodes.jcroyoaun.com/) - a tool for analyzing node output and identifying resource pressure issues.

## Architecture

This project consists of:

- **Go Application**: A chaos engineering tool that allows dynamic control of CPU, memory, latency, and health probe behavior
- **Kubernetes Deployments**: Four deployments showcasing different QoS classes
- **Terraform Infrastructure**: EKS cluster setup with Karpenter for node autoscaling
- **Ingress Configuration**: ALB-based routing with subdomain access

## QoS Class Deployments

### 1. Guaranteed (Most Protected)
**Endpoint**: `https://guaranteed.resourcehog.jcroyoaun.com`

- **Configuration**: `requests == limits`
- **Resources**: 1024Mi memory, 200m CPU
- **Replicas**: 3
- **Eviction Priority**: Last (most protected)
- **Use Case**: Critical production workloads requiring guaranteed resources

### 2. Burstable (Middle Priority)
**Endpoint**: `https://burstable.resourcehog.jcroyoaun.com`

- **Configuration**: `requests < limits` (4x overcommit)
- **Resources**: 128Mi-512Mi memory, 100m-500m CPU
- **Replicas**: 2
- **Eviction Priority**: Middle
- **Use Case**: Applications that occasionally need more resources than baseline

### 3. BestEffort (First to Evict)
**Endpoint**: `https://besteffort.resourcehog.jcroyoaun.com`

- **Configuration**: No resource requests or limits
- **Resources**: Unbounded
- **Replicas**: 2
- **Eviction Priority**: First (gets evicted first under pressure)
- **Use Case**: Low-priority batch jobs or development workloads

### 4. Troublemaker (Chaos Agent)
**Endpoints**: 
- `https://resourcehog.jcroyoaun.com`
- `https://chaos.resourcehog.jcroyoaun.com`

- **Configuration**: Massive overcommit (62x on memory)
- **Resources**: 345Mi-5Gi memory, 100m-3000m CPU
- **Replicas**: 1
- **Eviction Priority**: Middle (burstable class)
- **Use Case**: Testing cluster resilience and eviction behavior

## API Endpoints

All deployments expose the same HTTP API on port 8080.

### Control Endpoints

#### Set Memory Consumption
```bash
curl -X POST https://guaranteed.resourcehog.jcroyoaun.com/api/memory \
  -H "Content-Type: application/json" \
  -d '{"mb": 512, "ramp_seconds": 30}'
```

**Parameters:**
- `mb`: Target memory allocation in megabytes
- `ramp_seconds`: Time to gradually allocate memory (0 for immediate)

**Example Response:**
```json
{
  "status": "ok",
  "memory_mb": 512,
  "ramp_seconds": 30
}
```

#### Set CPU Consumption
```bash
curl -X POST https://burstable.resourcehog.jcroyoaun.com/api/cpu \
  -H "Content-Type: application/json" \
  -d '{"millicores": 250, "pattern": "sine"}'
```

**Parameters:**
- `millicores`: Target CPU usage (1000 = 1 core)
- `pattern`: Usage pattern - `constant`, `sine`, `spike`, or `ramp`

**Example Response:**
```json
{
  "status": "ok",
  "millicores": 250,
  "pattern": "sine"
}
```

#### Inject Latency
```bash
curl -X POST https://besteffort.resourcehog.jcroyoaun.com/api/latency \
  -H "Content-Type: application/json" \
  -d '{"ms": 500, "jitter_ms": 100}'
```

**Parameters:**
- `ms`: Base latency in milliseconds
- `jitter_ms`: Random jitter to add (0-N milliseconds)

**Example Response:**
```json
{
  "status": "ok",
  "latency_ms": 500,
  "jitter_ms": 100
}
```

#### Inject Errors
```bash
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/errors \
  -H "Content-Type: application/json" \
  -d '{"rate": 0.2}'
```

**Parameters:**
- `rate`: Error rate (0.0 to 1.0, where 0.2 = 20% error rate)

**Example Response:**
```json
{
  "status": "ok",
  "error_rate": 0.2
}
```

#### Control Health Probes
```bash
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/health \
  -H "Content-Type: application/json" \
  -d '{"ready": false, "live": true}'
```

**Parameters:**
- `ready`: Set readiness probe state (optional)
- `live`: Set liveness probe state (optional)

**Example Response:**
```json
{
  "status": "ok"
}
```

### Observability Endpoints

#### Get Status
```bash
curl https://guaranteed.resourcehog.jcroyoaun.com/api/status
```

**Example Response:**
```json
{
  "memory": {
    "target_mb": 512,
    "actual_mb": 512,
    "ramp_seconds": 30,
    "progress": 1.0
  },
  "cpu": {
    "target_millicores": 250,
    "pattern": "constant",
    "actual_percent": 25.0
  },
  "latency": {
    "ms": 0,
    "jitter_ms": 0
  },
  "error_rate": 0,
  "health": {
    "ready": true,
    "live": true
  },
  "goroutines": 12
}
```

#### Health Checks
```bash
# Liveness probe
curl https://guaranteed.resourcehog.jcroyoaun.com/healthz

# Readiness probe (respects latency and error injection)
curl https://guaranteed.resourcehog.jcroyoaun.com/readyz
```

#### Prometheus Metrics
```bash
curl https://guaranteed.resourcehog.jcroyoaun.com/metrics
```

**Example Output:**
```
# HELP resource_hog_goroutines Number of goroutines
# TYPE resource_hog_goroutines gauge
resource_hog_goroutines 12
# HELP resource_hog_memory_target_mb Target memory in MB
# TYPE resource_hog_memory_target_mb gauge
resource_hog_memory_target_mb 512
...
```

#### Expvar Debug Metrics
```bash
curl https://guaranteed.resourcehog.jcroyoaun.com/debug/vars
```

## Testing Scenarios

### Scenario 1: Test Memory Pressure
Allocate memory across different QoS classes to observe eviction behavior:

```bash
# Allocate memory on BestEffort (should evict first)
curl -X POST https://besteffort.resourcehog.jcroyoaun.com/api/memory \
  -d '{"mb": 2048, "ramp_seconds": 0}'

# Allocate memory on Burstable
curl -X POST https://burstable.resourcehog.jcroyoaun.com/api/memory \
  -d '{"mb": 1024, "ramp_seconds": 0}'

# Try to exceed guaranteed limits (should be killed by OOM)
curl -X POST https://guaranteed.resourcehog.jcroyoaun.com/api/memory \
  -d '{"mb": 2048, "ramp_seconds": 0}'
```

### Scenario 2: CPU Load Patterns
Test different CPU consumption patterns:

```bash
# Constant load
curl -X POST https://guaranteed.resourcehog.jcroyoaun.com/api/cpu \
  -d '{"millicores": 500, "pattern": "constant"}'

# Sine wave (oscillating load)
curl -X POST https://burstable.resourcehog.jcroyoaun.com/api/cpu \
  -d '{"millicores": 300, "pattern": "sine"}'

# Spike pattern (intermittent bursts)
curl -X POST https://besteffort.resourcehog.jcroyoaun.com/api/cpu \
  -d '{"millicores": 800, "pattern": "spike"}'
```

### Scenario 3: Chaos Testing
Simulate failures and observe Kubernetes recovery:

```bash
# Make pod fail readiness checks
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/health \
  -d '{"ready": false}'

# Inject 50% error rate
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/errors \
  -d '{"rate": 0.5}'

# Add latency to simulate slow responses
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/latency \
  -d '{"ms": 2000, "jitter_ms": 500}'

# Kill pod with liveness failure
curl -X POST https://chaos.resourcehog.jcroyoaun.com/api/health \
  -d '{"live": false}'
```

### Scenario 4: The Troublemaker
Test cluster resilience with massive resource consumption:

```bash
# Trigger extreme memory allocation (watch for eviction)
curl -X POST https://resourcehog.jcroyoaun.com/api/memory \
  -d '{"mb": 4096, "ramp_seconds": 10}'

# Max out CPU allocation
curl -X POST https://resourcehog.jcroyoaun.com/api/cpu \
  -d '{"millicores": 2500, "pattern": "spike"}'

# Monitor status
watch -n 1 'curl -s https://resourcehog.jcroyoaun.com/api/status | jq'
```

## Deployment

### Prerequisites

- Terraform >= 1.0
- kubectl configured with cluster access
- Docker for building the application
- AWS CLI configured

### Infrastructure Setup

```bash
cd infra
terraform init
terraform plan
terraform apply
```

This provisions:
- EKS cluster
- Karpenter for node autoscaling
- VPC and networking
- IAM roles and policies

### Application Deployment

```bash
cd k8s
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-guaranteed.yaml
kubectl apply -f 02-burstable.yaml
kubectl apply -f 03-besteffort.yaml
kubectl apply -f 04-troublemaker.yaml
kubectl apply -f 05-ingress.yaml
```

### Verify Deployment

```bash
# Check pod status
kubectl get pods -n qos-demo

# Verify QoS classes
kubectl get pods -n qos-demo -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass

# Check services
kubectl get svc -n qos-demo

# View ingress
kubectl get ingress -n qos-demo
```

## Monitoring

### Watch Pod Status
```bash
kubectl get pods -n qos-demo -w
```

### View Pod Resource Usage
```bash
kubectl top pods -n qos-demo
```

### Check Pod Events
```bash
kubectl get events -n qos-demo --sort-by='.lastTimestamp'
```

### View Logs
```bash
# Guaranteed pods
kubectl logs -n qos-demo -l qos=guaranteed -f

# Burstable pods
kubectl logs -n qos-demo -l qos=burstable -f

# BestEffort pods
kubectl logs -n qos-demo -l qos=besteffort -f

# Troublemaker
kubectl logs -n qos-demo -l role=troublemaker -f
```

## Project Structure

```
003/
├── app/
│   ├── Dockerfile           # Container image definition
│   └── main.go             # Go application source
├── infra/
│   ├── main.tf             # Terraform main configuration
│   ├── locals.tf           # Local variables
│   ├── outputs.tf          # Terraform outputs
│   ├── providers.tf        # Provider configuration
│   └── manifests/          # Karpenter configuration templates
├── k8s/
│   ├── 00-namespace.yaml   # qos-demo namespace
│   ├── 01-guaranteed.yaml  # Guaranteed QoS deployment
│   ├── 02-burstable.yaml   # Burstable QoS deployment
│   ├── 03-besteffort.yaml  # BestEffort QoS deployment
│   ├── 04-troublemaker.yaml # Chaos agent deployment
│   └── 05-ingress.yaml     # ALB ingress configuration
└── README.md               # This file
```

## Key Concepts

### Kubernetes QoS Classes

1. **Guaranteed**: Pods with `requests == limits` for all resources. These pods have the highest priority and are last to be evicted.

2. **Burstable**: Pods with `requests < limits` or only requests set. Middle priority for eviction.

3. **BestEffort**: Pods with no resource requests or limits. First to be evicted under resource pressure.

### Eviction Behavior

When a node runs out of resources (memory or disk), Kubernetes evicts pods in this order:
1. BestEffort pods (lowest priority)
2. Burstable pods exceeding requests
3. Burstable pods within requests
4. Guaranteed pods (only if system needs to reclaim resources)

### Resource Consumption Patterns

The application supports different CPU consumption patterns:
- **constant**: Steady CPU usage
- **sine**: Oscillating load (full cycle every 60 seconds)
- **spike**: Intermittent bursts (2 seconds on, 8 seconds low, repeating)
- **ramp**: Gradual increase (not yet implemented in this version)

## Troubleshooting

### Pods Not Starting
```bash
kubectl describe pod -n qos-demo <pod-name>
```

### Ingress Not Working
```bash
kubectl describe ingress -n qos-demo resource-hog-ingress
```

### DNS Not Resolving
Verify External DNS annotations and Route53 records:
```bash
nslookup guaranteed.resourcehog.jcroyoaun.com
```

### OOMKilled Pods
This is expected behavior when pods exceed memory limits:
```bash
kubectl get pods -n qos-demo -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[0].lastState.terminated.reason}{"\n"}{end}'
```

## Cleanup

```bash
# Delete Kubernetes resources
kubectl delete namespace qos-demo

# Destroy infrastructure
cd infra
terraform destroy
```

## License

This is a tutorial project for demonstration purposes.

