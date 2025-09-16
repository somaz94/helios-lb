# Helios Load Balancer

![Top Language](https://img.shields.io/github/languages/top/somaz94/helios-lb?color=green&logo=go&logoColor=b)
![helios-lb](https://img.shields.io/github/v/tag/somaz94/helios-lb?label=helios-lb&logo=kubernetes&logoColor=white)

Helios Load Balancer is a Kubernetes controller that provides load balancing functionality for bare metal Kubernetes clusters, similar to MetalLB. It automatically assigns IP addresses to LoadBalancer services and manages the network configuration.

<br/>

## Features

- Automatic IP address allocation for LoadBalancer services
- Support for IP ranges in CIDR or range format
- Multiple load balancing methods (Round Robin, Least Connection, Weighted Round Robin, IP Hash, Random)
- ARP-based layer 2 mode
- Configurable network interface
- Customizable ARP announcement intervals
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

```bash
kubectl apply -f https://raw.githubusercontent.com/somaz94/helios-lb/main/release/install.yaml
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
  method: RoundRobin  # Currently only RoundRobin is supported
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

- `ipRange`: IP range for allocation (required) - supports both range format (192.168.1.100-192.168.1.200) and CIDR format (192.168.1.0/24)
- `method`: Load balancing method (RoundRobin, LeastConnection, WeightedRoundRobin, IPHash, RandomSelection)
- `ports`: Port configuration for the service (default: 80)
- `protocol`: Protocol type (default: TCP)
- `healthCheckInterval`: Health check interval in seconds (default: 5)
- `metricsEnabled`: Enable or disable metrics collection (default: true)

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

Helios-LB supports multiple load balancing algorithms:

1. **Round Robin (Default)**
   ```yaml
   spec:
     method: RoundRobin
   ```
   - Distributes requests sequentially across all healthy backends
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
     weights:
       - serviceName: "service1"
         weight: 3
       - serviceName: "service2"
         weight: 1
   ```
   - Like Round Robin but with weighted distribution
   - Higher weight receives proportionally more traffic

4. **IP Hash**
   ```yaml
   spec:
     method: IPHash
   ```
   - Consistently maps client IPs to same backend
   - Useful for session persistence

5. **Random Selection**
   ```yaml
   spec:
     method: RandomSelection
   ```
   - Randomly selects from healthy backends
   - Simple but effective for even distribution

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
kubectl delete -f https://raw.githubusercontent.com/somaz94/helios-lb/main/release/install.yaml
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
make golangci-lint  # v1.61.0
```

Manual installation locations:
- All tools will be installed in `./bin` directory
- Specific versions:
  - controller-gen v0.16.4
  - kustomize v5.5.0
  - setup-envtest v0.19.0
  - golangci-lint v1.61.0

Note: The binary directory (`./bin`) is git-ignored and will be created when needed.

<br/>

## Contributing

Issues and pull requests are welcome.

<br/>

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

