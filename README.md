# Helios Load Balancer

![Top Language](https://img.shields.io/github/languages/top/somaz94/helios-lb?color=green&logo=go&logoColor=b)
![helios-lb](https://img.shields.io/github/v/tag/somaz94/helios-lb?label=helios-lb&logo=kubernetes&logoColor=white)

Helios Load Balancer is a Kubernetes controller that provides load balancing functionality for bare metal Kubernetes clusters, similar to MetalLB. It automatically assigns IP addresses to LoadBalancer services and manages the network configuration.

## Features

- Automatic IP address allocation for LoadBalancer services
- Support for IP ranges in CIDR or range format
- RoundRobin load balancing method
- ARP-based layer 2 mode
- Configurable network interface
- Customizable ARP announcement intervals
- Prometheus metrics support
- Status monitoring and reporting

## How it Works

The controller:
1. Watches for services of type LoadBalancer
2. Allocates IP addresses from configured IP ranges
3. Configures network interfaces with virtual IPs
4. Manages ARP announcements for layer 2 connectivity
5. Updates service status with allocated external IPs

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

## Installation

```bash
kubectl apply -f https://raw.githubusercontent.com/somaz94/helios-lb/main/config/samples/install.yaml
```

## Usage

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
curl -o nginx-test.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/config/samples/balancer_v1_heliosconfig.yaml

# port: 80,443 & protocol: tcp
curl -o nginx-test-port.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/config/samples/balancer_v1_heliosconfig_port.yaml
```

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
  loadBalancerIP: "10.10.10.65"
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: nginx-test

# Download the sample yaml file
curl -o nginx-test.yaml https://raw.githubusercontent.com/somaz94/helios-lb/main/config/samples/nginx-test.yaml
```

### Verification

Check the service status:
```bash
kubectl get svc nginx-test
```

You should see an external IP assigned from your configured IP range.

## Configuration

### HeliosConfig Options

- `ipRange`: IP range for allocation (required) - supports both range format (192.168.1.100-192.168.1.200) and CIDR format (192.168.1.0/24)
- `method`: Load balancing method (currently only supports RoundRobin)
- `ports`: Port configuration for the service (default: 80)
- `protocol`: Protocol type (default: TCP)
- `healthCheckInterval`: Health check interval in seconds (default: 5)
- `metricsEnabled`: Enable or disable metrics collection (default: true)

### System Ports

The controller uses the following system ports:
- Health probe endpoint: `:9082`
- Metrics endpoint (if enabled): `:8443`

These ports should be available when running the controller with hostNetwork enabled.

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

## Cleanup

1. Delete the LoadBalancer service:
```bash
kubectl delete svc nginx-test
```

2. Delete the HeliosConfig:
```bash
kubectl delete heliosconfig heliosconfig-sample
```

3. Remove the controller:
```bash
kubectl delete -f https://raw.githubusercontent.com/somaz94/helios-lb/main/config/samples/install.yaml
```

## Contributing

Issues and pull requests are welcome.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
