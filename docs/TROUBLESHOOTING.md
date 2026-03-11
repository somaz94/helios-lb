# Troubleshooting

<br/>

## Helm Test Issues

### `UPGRADE FAILED: "release-name" has no deployed releases`

A previous failed install/uninstall left a stuck release.

```bash
# Check stuck releases
helm list -a --all-namespaces | grep <release-name>

# Force cleanup
helm uninstall <release-name> --no-hooks
kubectl delete ns helios-lb-system --ignore-not-found
```

### CRD cleanup hook fails: `BackoffLimitExceeded`

The cleanup job's ServiceAccount lacks `apiextensions.k8s.io` CRD delete permission.

```bash
# Force uninstall without hooks
helm uninstall <release-name> --no-hooks

# Manually delete CRD if needed
kubectl delete crd heliosconfigs.balancer.helios.dev --ignore-not-found

# Delete stuck job
kubectl delete job -n helios-lb-system -l app.kubernetes.io/name=helios-lb --ignore-not-found
```

### Helm uninstall hangs

The pre-delete hook job is failing repeatedly.

```bash
# Cancel and force uninstall
helm uninstall <release-name> --no-hooks

# Clean up namespace
kubectl delete ns helios-lb-system --ignore-not-found
```

<br/>

## Controller Issues

### Controller pod is CrashLoopBackOff

```bash
# Check logs
kubectl logs -n helios-lb-system deployment/helios-lb-controller-manager --previous

# Check events
kubectl describe pod -n helios-lb-system -l control-plane=controller-manager
```

Common causes:
- CRD not installed: Run `make install` or reinstall Helm chart
- RBAC permission denied: Check ClusterRole and ClusterRoleBinding
- Port conflict: Metrics (8443) or health probe (9082) port already in use
- Network interface not available for ARP announcements

### CRD not found

```bash
# Verify CRD exists
kubectl get crd heliosconfigs.balancer.helios.dev

# Reinstall CRDs
make install
```

### Service external IP not assigned

```bash
# Check HeliosConfig status
kubectl get heliosconfig -o wide

# Verify HeliosConfig is Active
kubectl get heliosconfig <name> -o jsonpath='{.status.phase}'

# Check controller logs
kubectl logs -n helios-lb-system deployment/helios-lb-controller-manager -f

# Verify loadBalancerClass is set
kubectl get svc <name> -o jsonpath='{.spec.loadBalancerClass}'
# Should be: helios-lb
```

### IP allocation fails

- Verify IP range is valid and not conflicting with existing network
- Ensure the IP is reachable from the cluster network
- Check if the IP is already allocated to another service

```bash
# Check allocated IPs
kubectl get svc -A -o jsonpath='{range .items[?(@.spec.type=="LoadBalancer")]}{.metadata.name}: {.status.loadBalancer.ingress[0].ip}{"\n"}{end}'
```

### ARP announcements not working

- Ensure `hostNetwork: true` is configured for the controller
- Verify the network interface exists on the node
- Check if another load balancer (e.g., MetalLB) is conflicting

```bash
# Check ARP entries on the node
arping -c 3 <external-ip>
```

### Conflict with MetalLB

Ensure IP ranges don't overlap between Helios-LB and MetalLB. Use `loadBalancerClass: helios-lb` on services meant for Helios-LB.

<br/>

## CI/CD Issues

### `git push` rejected (remote ahead)

Workflow-generated commits (CHANGELOG.md) can make remote ahead.

```bash
git pull --rebase origin main
git push origin main
```

### Release workflow: `GITHUB_TOKEN` doesn't trigger other workflows

This is expected. Use `PAT_TOKEN` for operations that need to trigger downstream workflows.

### Dependabot PR merge fails: OAuth token lacks `workflow` scope

Dependabot PRs that modify `.github/workflows/` files need the `workflow` scope. Merge these manually via GitHub web UI.

<br/>

## Build Issues

### `make manifests generate` shows diff in CI

Generated files are out of date. Run locally and commit:

```bash
make manifests generate
git add config/ api/
git commit -m "chore: update generated manifests"
```
