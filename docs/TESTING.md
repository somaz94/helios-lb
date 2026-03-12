# Helios LB - Test Guide

<br/>

## Prerequisites

- Kubernetes cluster (Kind, Minikube, EKS, GKE, etc.)
- `kubectl` configured
- No conflicting LB controller (MetalLB, Cilium LB) on the same IP range
- `make`, `helm`, `go` installed

<br/>

## 1. Unit Tests

```bash
make test
```

Runs all unit and envtest-based tests with coverage report (`cover.out`).

<br/>

## 2. Integration Test (Automated)

Run all sample tests automatically against a live cluster:

```bash
make test-integration
```

The script will:
- Auto-install CRD if not found (`make install`)
- Auto-deploy controller if not running (`make deploy`)
- Check for IP conflicts before testing (ping, arping, ARP cache, K8s services)
- Apply sample HeliosConfig resources and verify IP allocation

### Custom Test IPs

```bash
# Override test IPs via environment variables
TEST_IP=192.168.1.100 TEST_IP2=192.168.1.101 make test-integration

# Use an IP range
TEST_IP_RANGE="192.168.1.100-192.168.1.110" make test-integration
```

<br/>

## 3. IP Scan Utility

Before running integration tests, scan for available IPs:

```bash
make find-free-ip
```

Options:

```bash
# Custom range and count
make find-free-ip START_IP=192.168.1.100 END_IP=192.168.1.120 COUNT=3

# Show usage help
make find-free-ip HELP=1
```

Scan methods: ping (ICMP), arping (ARP broadcast), ARP cache, K8s service ingress check.

<br/>

## 4. Helm Chart Test (Automated)

Run Helm chart tests (lint, template, install, sync tests, uninstall):

```bash
make test-helm
```

Test coverage:
- Helm lint, template render
- Helm install & release verification
- Controller pod, CRD, RBAC, Service verification
- HeliosConfig CR tests (basic, port, weight)
- IP allocation and service status checks
- Helm uninstall & CRD cleanup hook verification

<br/>

## 5. E2E Tests

Requires a Kind cluster:

```bash
make test-e2e
```

<br/>

## 6. Manual Deploy Test

<br/>

### Step 1: Deploy Controller (CRD + RBAC + Controller)

```bash
make deploy
```

<br/>

### Step 2: Verify Controller is Running

```bash
kubectl get pods -n helios-lb-system
```

<br/>

### Step 3: Deploy Test Service

```bash
kubectl apply -f config/samples/nginx-test.yaml
```

Verify the service is created:

```bash
kubectl get svc nginx-test
```

---

## 7. HeliosConfig Tests

<br/>

### Test A: Basic IP Allocation

```bash
kubectl apply -f config/samples/balancer_v1_heliosconfig.yaml
```

Check:

```bash
kubectl get heliosconfigs
kubectl get svc -l type=LoadBalancer
```

Cleanup:

```bash
kubectl delete heliosconfigs --all
```

<br/>

### Test B: Port Configuration

```bash
kubectl apply -f config/samples/balancer_v1_heliosconfig_port.yaml
```

Check:

```bash
kubectl get heliosconfigs
kubectl describe heliosconfig heliosconfig-port
```

Cleanup:

```bash
kubectl delete heliosconfigs --all
```

<br/>

### Test C: Weighted Load Balancing

```bash
kubectl apply -f config/samples/balancer_v1_heliosconfig_weight.yaml
```

Check:

```bash
kubectl get heliosconfigs
kubectl describe heliosconfig heliosconfig-weight
```

Cleanup:

```bash
kubectl delete heliosconfigs --all
```

---

## 8. LB Controller Conflict Check

Helios LB automatically skips services managed by other LB controllers (MetalLB, Cilium LB, etc.) via `loadBalancerClass` filtering.

Verify no conflicts:

```bash
# Check for other LB controllers
kubectl get pods -A | grep -E "metallb|cilium"

# Check loadBalancerClass on services
kubectl get svc -A -o jsonpath='{range .items[?(@.spec.type=="LoadBalancer")]}{.metadata.name}{"\t"}{.spec.loadBalancerClass}{"\n"}{end}'
```

---

## 9. Finalizer / Deletion Test

Verify that deleting a HeliosConfig releases allocated IPs:

```bash
# Apply a config
kubectl apply -f config/samples/balancer_v1_heliosconfig.yaml

# Check IP is allocated
kubectl get heliosconfigs -o jsonpath='{.items[0].status.allocatedIPs}'

# Delete the config
kubectl delete heliosconfigs --all

# IP should be released (service ingress cleared)
kubectl get svc nginx-test -o jsonpath='{.status.loadBalancer}'
```

---

## 10. Full Cleanup

```bash
# Remove all test resources
kubectl delete -f config/samples/nginx-test.yaml --ignore-not-found
kubectl delete heliosconfigs --all --ignore-not-found

# Undeploy controller (removes CRD + RBAC + Controller)
make undeploy
```

---

## Sample Files

| File | Type | Description |
|------|------|-------------|
| `balancer_v1_heliosconfig.yaml` | Basic | Single IP allocation with RoundRobin |
| `balancer_v1_heliosconfig_port.yaml` | Port | IP allocation with port configuration |
| `balancer_v1_heliosconfig_weight.yaml` | Weight | Weighted load balancing configuration |
| `nginx-test.yaml` | Test | Nginx deployment + LoadBalancer service |
