# Helios Load Balancer

![Top Language](https://img.shields.io/github/languages/top/somaz94/helios-lb?color=green&logo=go&logoColor=b)
![helios-lb](https://img.shields.io/github/v/tag/somaz94/helios-lb?label=helios-lb&logo=kubernetes&logoColor=white)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/somaz94/helios-lb)](https://goreportcard.com/report/github.com/somaz94/helios-lb)
![Docker Pulls](https://img.shields.io/docker/pulls/somaz940/helios-lb?logo=docker&logoColor=white)
![GitHub Release](https://img.shields.io/github/release/somaz94/helios-lb?logo=github)
![GitHub Stars](https://img.shields.io/github/stars/somaz94/helios-lb?style=social)

Helios Load Balancer is a Kubernetes controller that provides load balancing functionality for bare metal Kubernetes clusters, similar to MetalLB. It automatically assigns IP addresses to LoadBalancer services and manages the network configuration.

<br/>

## Features

![IP Allocation](https://img.shields.io/badge/IP_Allocation-blue?logo=kubernetes&logoColor=white)
![CIDR Range](https://img.shields.io/badge/CIDR_Range-blue?logo=kubernetes&logoColor=white)
![Round Robin](https://img.shields.io/badge/Round_Robin-green?logo=kubernetes&logoColor=white)
![Least Connection](https://img.shields.io/badge/Least_Connection-green?logo=kubernetes&logoColor=white)
![Weighted Round Robin](https://img.shields.io/badge/Weighted_Round_Robin-green?logo=kubernetes&logoColor=white)
![IP Hash](https://img.shields.io/badge/IP_Hash-green?logo=kubernetes&logoColor=white)
![Random](https://img.shields.io/badge/Random-green?logo=kubernetes&logoColor=white)
![ARP Layer2](https://img.shields.io/badge/ARP_Layer2-orange?logo=kubernetes&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus_Metrics-E6522C?logo=prometheus&logoColor=white)
![Bare Metal](https://img.shields.io/badge/Bare_Metal-326CE5?logo=kubernetes&logoColor=white)

- Automatic IP address allocation for LoadBalancer services
- Support for IP ranges in CIDR, range, or single IP format
- Multiple load balancing methods (RoundRobin, LeastConnection, WeightedRoundRobin, IPHash, Random)
- Multiple HeliosConfig resources per cluster with independent IP ranges
- Configurable health checks (TCP/HTTP, custom timeout and interval)
- ARP-based layer 2 mode
- Pluggable algorithm interface for custom load balancing strategies
- Prometheus metrics support
- Status monitoring and reporting

<br/>

## How it Works

The controller:
1. Watches for services of type LoadBalancer
2. Allocates IP addresses from configured IP ranges
3. Configures network interfaces with virtual IPs
4. Manages ARP announcements for layer 2 connectivity
5. Updates service status with allocated external IPs

<br/>

## Coexistence with MetalLB

Helios Load Balancer can coexist with MetalLB in the same cluster if properly configured:

1. Configure different IP ranges for each load balancer:
   ```yaml
   # MetalLB IPAddressPool
   apiVersion: metallb.io/v1beta1
   kind: IPAddressPool
   metadata:
     name: first-pool
     namespace: metallb-system
   spec:
     addresses:
     - 192.168.1.100-192.168.1.200  # MetalLB range
   ```

   ```yaml
   # Helios-LB HeliosConfig
   apiVersion: balancer.helios.dev/v1
   kind: HeliosConfig
   metadata:
     name: heliosconfig-sample
   spec:
     ipRange: "10.10.10.65"  # Helios-LB range
   ```

2. When creating a LoadBalancer service, specify which load balancer should handle it by using the appropriate IP range:
   ```yaml
   # Service using MetalLB
   apiVersion: v1
   kind: Service
   metadata:
     name: metallb-service
   spec:
     type: LoadBalancer
     loadBalancerIP: "192.168.1.100"  # MetalLB range
   ```

   ```yaml
   # Service using Helios-LB
   apiVersion: v1
   kind: Service
   metadata:
     name: helios-service
   spec:
     type: LoadBalancer
     loadBalancerIP: "10.10.10.65"  # Helios-LB range
   ```

This way, you can leverage both load balancers in your cluster, each managing its own IP range.

<br/>

## Installation

<br/>

### Prerequisites
- Kubernetes v1.16+
- kubectl v1.11.3+

<br/>

### Option 1: Helm (Recommended)

```bash
# Add the Helm repository
helm repo add helios-lb https://somaz94.github.io/helios-lb/helm-repo
helm repo update

# Install with default values
helm install helios-lb helios-lb/helios-lb

# Or install with custom values
helm install helios-lb helios-lb/helios-lb \
  --set image.tag=v0.2.6 \
  --namespace helios-lb-system --create-namespace
```

For full Helm chart options, see [Helm README](docs/HELM.md).

<br/>

### Option 2: kubectl apply (Quick Install)

```bash
kubectl apply -f https://raw.githubusercontent.com/somaz94/helios-lb/main/dist/install.yaml
```

<br/>

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/somaz94/helios-lb.git
cd helios-lb

# Install CRDs
make install

# Deploy the controller
make deploy IMG=somaz940/helios-lb:v0.2.6
```

<br/>

## Usage

<br/>

### 1. Create a HeliosConfig:

```yaml
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: heliosconfig-sample
spec:
  # IP range for virtual IP allocation (CIDR or range format) default port: 80 & default protocol: tcp
  ipRange: "10.10.10.65" # 192.168.1.100-192.168.1.200 or 192.168.1.100
  method: RoundRobin  # RoundRobin, LeastConnection, WeightedRoundRobin, IPHash, Random
  ports:              # Optional: default is 80
  - port: 80
  - port: 443
  protocol: TCP      # Optional: default is TCP

# Download the sample yaml file

# default port: 80 & default protocol: tcp
curl -o heliosconfig.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/release/examples/balancer_v1_heliosconfig.yaml

# port: 80,443 & protocol: tcp
curl -o heliosconfig-port.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/release/examples/balancer_v1_heliosconfig_port.yaml
```

<br/>

### 2. Deploy a service with type LoadBalancer:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  labels:
    app: nginx-test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx-test
  template:
    metadata:
      labels:
        app: nginx-test
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-test
spec:
  type: LoadBalancer
  loadBalancerClass: helios-lb
  loadBalancerIP: "10.10.10.65"
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: nginx-test


# Download the sample yaml file
curl -o nginx-test.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/release/examples/nginx-test.yaml
```

<br/>

### Verification

Check the service status:
```bash
kubectl get svc nginx-test
...
NAME         TYPE           CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
nginx-test   LoadBalancer   10.233.32.189   10.10.10.65   80:30396/TCP   2s

k get heliosconfigs
NAME                       PHASE    MESSAGE                     AGE
heliosconfig-sample-port   Active   IP allocated successfully   26s
```

You should see an external IP assigned from your configured IP range.

<br/>

## Configuration

<br/>

### HeliosConfig Options

- `ipRange`: IP range for allocation (required) - supports single IP (`192.168.1.100`), range format (`192.168.1.100-192.168.1.200`), and CIDR format (`192.168.1.0/24`)
- `method`: Load balancing method (`RoundRobin`, `LeastConnection`, `WeightedRoundRobin`, `IPHash`, `Random`)
- `ports`: Port configuration for the service (default: 80)
- `protocol`: Protocol type (default: TCP)
- `healthCheck`: Health check configuration (optional)
  - `enabled`: Enable/disable health checking (default: true)
  - `intervalSeconds`: Interval between health checks in seconds (default: 5, range: 1-300)
  - `timeoutMs`: Health check timeout in milliseconds (default: 1000, range: 1-30000)
  - `protocol`: Health check protocol - `TCP` or `HTTP` (default: TCP)
  - `httpPath`: HTTP path for HTTP health checks (e.g., `/healthz`)

<br/>

### System Ports

The controller uses the following system ports:
- Health probe endpoint: `:9082`
- Metrics endpoint (if enabled): `:8443`

These ports should be available when running the controller with hostNetwork enabled.

### Load Balancer Class

To ensure proper handling of services between MetalLB and Helios-LB, use the `loadBalancerClass` field:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-test
spec:
  type: LoadBalancer
  loadBalancerClass: helios-lb  # Specifies that Helios-LB should handle this service
  loadBalancerIP: "10.10.10.65"
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: nginx-test
```

The controller automatically adds the necessary annotations and specifications:
- Adds annotation: `balancer.helios.dev/load-balancer-class: helios-lb`
- Sets spec: `loadBalancerClass: helios-lb`

This ensures that:
1. MetalLB ignores services marked for Helios-LB
2. Helios-LB only processes services specifically marked for it
3. No conflicts occur between the two load balancers

<br/>

### Load Balancing Algorithms

Helios-LB supports the following load balancing algorithms:

1. **Round Robin (Default)**
   ```yaml
   spec:
     method: RoundRobin
   ```
   - Distributes requests sequentially across all backends
   - Ensures even distribution of traffic

2. **Least Connection**
   ```yaml
   spec:
     method: LeastConnection
   ```
   - Directs traffic to backend with fewest active connections
   - Ideal for long-lived connections

3. **Weighted Round Robin**
   ```yaml
   spec:
     method: WeightedRoundRobin
   ```
   - Distributes traffic based on backend weights
   - Higher-weight backends receive more traffic

4. **IP Hash**
   ```yaml
   spec:
     method: IPHash
   ```
   - Routes traffic based on client IP hash
   - Provides session persistence (same client → same backend)

5. **Random**
   ```yaml
   spec:
     method: Random
   ```
   - Randomly selects a healthy backend
   - Simple and effective for homogeneous backends

<br/>

## Troubleshooting

Common issues and solutions:

1. **IP allocation fails:**
   - Verify IP range configuration
   - Check network interface exists
   - Review controller logs

2. **Service external IP not assigned:**
   - Verify HeliosConfig exists and is valid
   - Check controller logs for errors
   - Verify network interface configuration

3. **Network connectivity issues:**
   - Check ARP announcements
   - Verify network interface configuration
   - Ensure IP range is valid for your network

<br/>

## Cleanup

1. Delete the LoadBalancer service:
```bash
kubectl delete -f https://raw.githubusercontent.com/somaz94/helios-lb/main/release/examples/nginx-test.yaml
```

2. Delete the HeliosConfig:
```bash
k get heliosconfig
kubectl delete heliosconfig <heliosconfig-name>
```

3. Remove the controller:
```bash
kubectl delete -f https://raw.githubusercontent.com/somaz94/helios-lb/main/dist/install.yaml
```

<br/>

## Development Setup

<br/>

### Install Required Tools

All required tools will be automatically downloaded to `./bin` directory when running:
```bash
make install-tools
```

Or you can install individual tools:
```bash
# Install controller-gen
make controller-gen  # v0.16.4

# Install kustomize
make kustomize      # v5.5.0

# Install setup-envtest
make envtest        # v0.19.0

# Install golangci-lint
make golangci-lint  # v2.1.6
```

Manual installation locations:
- All tools will be installed in `./bin` directory
- Specific versions:
  - controller-gen v0.16.4
  - kustomize v5.5.0
  - setup-envtest v0.19.0
  - golangci-lint v2.1.6

Note: The binary directory (`./bin`) is git-ignored and will be created when needed.

<br/>

## Documentation

| Document | Description |
|----------|-------------|
| [Helm Chart](docs/HELM.md) | Helm chart installation, configuration, and values reference |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and solutions |
| [Version Bump](docs/VERSION_BUMP.md) | Checklist for releasing a new version |
| [Contributing](CONTRIBUTING.md) | How to contribute to this project |

<br/>

## Contributing

Issues and pull requests are welcome.

<br/>

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

