# CLAUDE.md

<br/>

## Commit Guidelines

- Do not include `Co-Authored-By` lines in commit messages.
- Do not push to remote. Only commit. The user will push manually.

<br/>

## Project Structure

- Kubernetes operator built with controller-runtime (kubebuilder)
- CRD: `HeliosConfig` (apiGroup: `balancer.helios.dev/v1`)
- Bare-metal load balancer for Kubernetes (similar to MetalLB)
- Features: IP allocation (CIDR/range/single, IPv4/IPv6), multiple HeliosConfig support, load balancing (RoundRobin, LeastConnection, WeightedRoundRobin, IPHash, Random via Algorithm interface), per-service weights, namespace isolation, per-config quota (maxAllocations), configurable health checks (TCP/HTTP), validating webhook (IP range/port/overlap), ARP-based L2 mode, Prometheus metrics

<br/>

## Build & Test

```bash
make test                # Unit tests (envtest)
make test-e2e            # E2E tests (requires Kind cluster)
make test-integration    # Integration tests (uses make deploy, local source)
make test-helm           # Helm chart tests (uses local chart path)
make manifests generate  # Regenerate CRD and RBAC manifests
make lint                # Run golangci-lint
make bump-version VERSION=vX.Y.Z  # Bump version across all files
```

<br/>

## Key Directories

- `api/v1/` — CRD type definitions (HeliosConfig spec/status)
- `internal/controller/` — Reconciler logic
- `internal/loadbalancer/` — Load balancing algorithms, health checks, metrics
- `internal/network/` — IP allocation and network configuration
- `internal/metrics/` — Prometheus metrics
- `config/samples/` — Sample CR YAML files
- `helm/helios-lb/` — Helm chart
- `hack/` — Utility scripts (bump-version, find-free-ip, test-integration, test-helm)
- `docs/` — Documentation

<br/>

## Code Style

- Linter: `golangci-lint` — prefer `switch` over `if-else if` chains.
- Run `make lint` before committing.

<br/>

## Common Pitfalls

- **Helm CRD sync**: When adding/changing CRD fields, always copy from `config/crd/bases/` to `helm/helios-lb/crds/`.
- **Dockerfile ARG scope**: In multi-stage builds, ARG must be re-declared after each `FROM`.
- **Version bump**: `make bump-version` auto-updates all files (Makefile, Chart.yaml, values.yaml, README, docs, dist/install.yaml). No manual edits needed.
- **IP scanning**: Use `hack/find-free-ip.sh START_IP END_IP COUNT` to find available IPs before testing.

<br/>

## Release Workflow

1. `make bump-version VERSION=vX.Y.Z`
2. Commit & push
3. `make docker-buildx` (build & push image)
4. `git tag vX.Y.Z && git push origin vX.Y.Z`
5. Tag push auto-triggers: `release.yml`, `helm-release.yml`, `changelog-generator.yml`

<br/>

## Language

- Communicate with the user in Korean.
- All documentation and code comments must be written in English.
