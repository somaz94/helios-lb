
### Bug Fixes
- release.yml & changelog-generator.yml by @somaz94
- chagnelog-generator.yml by @somaz94
- changelog-generator.yml by @somaz94
- changelog-generator.yml, .gitignore, add: gitlab-mirror.yml by @somaz94
- test, test-e2e.yml by @somaz94
- metrics_test.go, network_test.go, health.go by @somaz94
- balancer_test.go, metrics.go by @somaz94
- balancer.go, balancer_test.go by @somaz94
- heliosconfig_controller_test.go by @somaz94
- test/e2e/e2e_test.go by @somaz94
- dynamically resolve tags in changelog and release workflows by @somaz94
- resolve helm-repo checkout conflict in helm-release workflow by @somaz94
- remove helm-repo before checkout to prevent untracked file conflict by @somaz94
- set GOTOOLCHAIN from go.mod to fix covdata errors by @somaz94
- rename crd-clenaup typo, add dedicated RBAC for cleanup hook, fix Chart version by @somaz94
- auto-install CRD and deploy controller in integration test by @somaz94
- add pod creation wait loop before kubectl wait in integration test by @somaz94
- resolve critical bugs and security issues by @somaz94
- sync State and Phase fields in HeliosConfig status by @somaz94
- force imagePullPolicy Always during integration tests by @somaz94
- clear service ingress on HeliosConfig deletion by @somaz94
- use IP range matching instead of string comparison for service filtering by @somaz94
- add trap for cleanup and undeploy on test exit by @somaz94
- show full undeploy output in integration test cleanup by @somaz94

### CI/CD
- use conventional commit message in changelog-generator workflow by @somaz94
- add helm chart release workflow for automated packaging by @somaz94
- add release notes categorization config by @somaz94
- migrate release workflow to git-cliff and softprops/action-gh-release by @somaz94
- add automation workflows and manifests verification by @somaz94

### Chore
- bump golang from 1.23 to 1.24 in the docker-minor group by @dependabot[bot]
- bump the go-minor group with 4 updates by @dependabot[bot]
- bump github.com/prometheus/client_golang by @dependabot[bot]
- bump the go-minor group with 3 updates by @dependabot[bot]
- bump the go-minor group with 3 updates by @dependabot[bot]
- bump the go-minor group with 2 updates by @dependabot[bot]
- bump golangci/golangci-lint-action from 6 to 7 by @dependabot[bot]
- bump sigs.k8s.io/controller-runtime in the go-minor group by @dependabot[bot]
- bump the go-minor group across 1 directory with 3 updates by @dependabot[bot]
- bump the go-minor group with 3 updates by @dependabot[bot]
- bump golangci/golangci-lint-action from 7 to 8 by @dependabot[bot]
- bump golang from 1.24 to 1.25 in the docker-minor group by @dependabot[bot]
- bump actions/checkout from 4 to 5 by @dependabot[bot]
- bump the go-minor group across 1 directory with 7 updates by @dependabot[bot]
- bump golangci/golangci-lint-action from 8 to 9 by @dependabot[bot]
- bump the go-minor group across 1 directory with 8 updates by @dependabot[bot]
- bump actions/setup-go from 5 to 6 by @dependabot[bot]
- workflows by @somaz94
- bump actions/checkout from 5 to 6 by @dependabot[bot]
- test, test-e2e by @somaz94
- bump the go-minor group with 5 updates by @dependabot[bot]
- bump the go-minor group with 4 updates by @dependabot[bot]
- bump golang from 1.25 to 1.26 in the docker-minor group by @dependabot[bot]
- bump the go-minor group across 1 directory with 6 updates by @dependabot[bot]
- bump the go-minor group across 1 directory with 4 updates by @dependabot[bot]
- bump sigs.k8s.io/controller-runtime in the go-minor group by @dependabot[bot]
- upgrade Go to 1.26, use go-version-file in CI workflows by @somaz94
- bump version to v0.2.6 by @somaz94
- remove hardcoded IPs from samples, add find-free-ip make target by @somaz94
- add --help option to find-free-ip and HELP=1 make target by @somaz94

### Documentation
- README.md by @somaz94
- consolidate documentation into docs/ directory by @somaz94
- add TROUBLESHOOTING.md and CONTRIBUTING.md, migrate golangci-lint to v2 by @somaz94
- docs/VERSION_BUMP.md by @somaz94
- add TESTING.md for test guide and procedures by @somaz94

### Features
- add test-integration and test-helm scripts with Makefile targets by @somaz94
- add loadBalancerClass filtering to avoid conflicts with other LB controllers by @somaz94
- add free IP scanner and enhance IP conflict detection by @somaz94

### Performance
- optimize Dockerfile with build cache and smaller binary by @somaz94

### Testing
- enhance controller test coverage from 65.4% to 95.1% by @somaz94
- enhance loadbalancer coverage from 88.5% to 99.4% by @somaz94

**Full Changelog**: https://github.com/somaz94/helios-lb/compare/v0.2.5...v0.2.6
