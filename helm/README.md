# Helios Load Balancer Helm Chart

## Introduction
This Helm chart installs Helios Load Balancer Controller on your Kubernetes cluster. The controller provides load balancing functionality with various methods including RoundRobin and WeightedRoundRobin.

## Prerequisites
- Kubernetes 1.16+
- Helm 3.0+

## Installing the Chart

Add the Helm repository:

```bash
helm repo add helios-lb https://somaz94.github.io/helios-lb/helm-repo
helm repo update
```

Install the chart:
```bash
helm install helios-lb helios-lb/helios-lb
```

To install with custom values:
```bash
helm install helios-lb helios-lb/helios-lb -f values.yaml
```

## Configuration

The following table lists the configurable parameters of the helios-lb chart and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `namespace` | Namespace where the controller will be installed | `helios-lb-system` |
| `image.repository` | Controller image repository | `somaz940/helios-lb` |
| `image.tag` | Controller image tag | `v0.2.4` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `resources.limits.cpu` | CPU resource limits | `500m` |
| `resources.limits.memory` | Memory resource limits | `128Mi` |
| `resources.requests.cpu` | CPU resource requests | `10m` |
| `resources.requests.memory` | Memory resource requests | `64Mi` |
| `controller.metrics.bindAddress` | Metrics bind address | `:8443` |
| `controller.health.bindAddress` | Health probe bind address | `:9082` |
| `controller.leaderElection.enabled` | Enable leader election | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `crds.create` | Create CRD resources | `true` |
| `serviceAccount.create` | Create ServiceAccount | `true` |
| `serviceAccount.name` | ServiceAccount name | `helios-lb-controller-manager` |
| `customresource.basic.enabled` | Enable basic load balancer configuration | `false` |
| `customresource.port.enabled` | Enable port-specific configuration | `false` |
| `customresource.weight.enabled` | Enable weighted load balancer configuration | `false` |

## Custom Resource Configuration

The chart supports creating different types of HeliosConfig resources during installation. You can enable and configure them in your values file:

### Basic Configuration
```yaml
customresource:
  basic:
    enabled: true
    name: "heliosconfig-basic"
    ipRange: "10.10.10.65"
    method: "RoundRobin"
```

### Port Configuration
```yaml
customresource:
  port:
    enabled: true
    name: "heliosconfig-port"
    ipRange: "10.10.10.65"
    method: "RoundRobin"
    ports:
      - port: 80
      - port: 443
```

### Weighted Configuration
```yaml
customresource:
  weight:
    enabled: true
    name: "heliosconfig-weight"
    ipRange: "10.10.10.65"
    ports:
      - port: 80
      - port: 443
    weights:
      - serviceName: "service1"
        weight: 3
      - serviceName: "service2"
        weight: 2
      - serviceName: "service3"
        weight: 1
```

Local install Method:
```bash
git clone https://github.com/somaz94/helios-lb.git
cd helios-lb
helm install helios-lb ./helm/helios-lb -f ./helm/helios-lb/values/basic-values.yaml
```

## Usage

After installing the chart, you can create a HeliosConfig resource to start load balancing:

```yaml
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: heliosconfig-sample
spec:
  ipRange: "10.10.10.65"
  method: RoundRobin
  ports:
    - port: 80
    - port: 443
```

## Uninstalling the Chart

To uninstall/delete the deployment:
```bash
helm delete helios-lb
```

## Upgrading the Chart

To upgrade the chart:
```bash
helm upgrade helios-lb helios-lb/helios-lb
```

## Troubleshooting

### Verify Installation
```bash
# Check if pods are running
kubectl get pods -n helios-lb-system

# Check controller logs
kubectl logs -n helios-lb-system -l control-plane=controller-manager -f
```

### Common Issues

1. **CRD not installed**
   - Ensure CRDs are installed:
     ```bash
     kubectl get crd heliosconfigs.balancer.helios.dev
     ```

2. **Permission Issues**
   - Verify RBAC settings:
     ```bash
     kubectl get clusterrole,clusterrolebinding -l app.kubernetes.io/name=helios-lb
     ```

3. **Pod not starting**
   - Check pod events:
     ```bash
     kubectl describe pod -n helios-lb-system -l control-plane=controller-manager
     ```

## Support

For support, please check:
- [Documentation](https://github.com/somaz94/helios-lb)
- [Issues](https://github.com/somaz94/helios-lb/issues)
