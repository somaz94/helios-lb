# Changelog

All notable changes to this project will be documented in this file.

## Unreleased (2026-03-23)

### Features

- implement Status Conditions for standard Kubernetes observability ([65f49c9](https://github.com/somaz94/helios-lb/commit/65f49c99e8cd126f66442e253364fd0638068f39))
- add Kubernetes Events recording for IP lifecycle ([17dd568](https://github.com/somaz94/helios-lb/commit/17dd56829e60d09191edc0fcf6e03bc231c1977a))

### Bug Fixes

- restore Chart.yaml before gh-pages checkout in helm-release workflow ([a303098](https://github.com/somaz94/helios-lb/commit/a303098359d7976fa0088cec7bf93185fff3e909))
- prevent duplicate quote in bump-version.sh values.yaml sed pattern ([9c7acfa](https://github.com/somaz94/helios-lb/commit/9c7acfa24ec95460f2a49fbab4189ba9211c2993))
- remove duplicate quote in values.yaml image tag ([b25c770](https://github.com/somaz94/helios-lb/commit/b25c770b38f59a9dbdd2b9c71439956cf977bdae))

### Code Refactoring

- extract IP allocation and service filtering from controller ([7a5fc5e](https://github.com/somaz94/helios-lb/commit/7a5fc5e1beadf0616b87989a36611e479326c887))
- add custom error types for retryable/permanent error handling ([9ec9aa4](https://github.com/somaz94/helios-lb/commit/9ec9aa49631ea7a0e8876a43aa74d75e96c1a240))
- apply structured logging with consistent key-value patterns ([fcffdb0](https://github.com/somaz94/helios-lb/commit/fcffdb042354391c82a0268f5f3dc61e999566bb))

### Continuous Integration

- restrict push trigger to main branch to prevent duplicate CI runs ([924fbb8](https://github.com/somaz94/helios-lb/commit/924fbb8a08fbb4243fde1f388b5be5e61b120eaa))

### Chores

- regenerate RBAC role with events permission ([6d7231a](https://github.com/somaz94/helios-lb/commit/6d7231a97dd505bb9d0fa3b7e2b20627008c1546))
- regenerate RBAC role with events permission ([948cecb](https://github.com/somaz94/helios-lb/commit/948cecbc6fa1fc843dfb8b23beb1928389c9baf0))

### Contributors

- somaz

<br/>

## [v0.3.0](https://github.com/somaz94/helios-lb/compare/v0.2.6...v0.3.0) (2026-03-23)

### Features

- add benchmarks, reconciliation metrics, and cert-manager support ([9b4fc4c](https://github.com/somaz94/helios-lb/commit/9b4fc4c5508d5ee155b65b46ad893c21accd4012))
- add IPv6 support, namespace isolation, weights, validating webhook ([bfeb6d2](https://github.com/somaz94/helios-lb/commit/bfeb6d2f5adc1bc60e645aafce424af9b3753a5b))
- add multi-config support, CIDR, algorithm interface, configurable health checks ([db87d19](https://github.com/somaz94/helios-lb/commit/db87d19aa06f8280dcfa021cbe3a5b412ef5da15))
- add CODEOWNERS ([a17c335](https://github.com/somaz94/helios-lb/commit/a17c335d38bdb429577a6b18de052673d262c09c))

### Bug Fixes

- make webhook registration conditional to fix e2e test failure ([cd33b07](https://github.com/somaz94/helios-lb/commit/cd33b075ef97ff55a237ddd5ba3b9da781daf3fb))
- use GITHUB_TOKEN for dependabot auto merge ([25291d2](https://github.com/somaz94/helios-lb/commit/25291d220886deeb9273c9a8cb62054c390b3bd9))
- re-declare ARG in final stage for OCI labels ([0ee1624](https://github.com/somaz94/helios-lb/commit/0ee16243d7dab9ac90b00c29512f84f6f35b9291))
- set imagePullPolicy=Always in helm test script ([321a768](https://github.com/somaz94/helios-lb/commit/321a768276b7cfa88b627b6971c6ee96550578a1))
- skip major version tag deletion on first release ([f16e83c](https://github.com/somaz94/helios-lb/commit/f16e83ce70dac9954b5003f567c8aeab9c3d41ba))
- pin kubectl image version in CRD cleanup hook ([f246591](https://github.com/somaz94/helios-lb/commit/f246591be8233889f6868f03130a063bbfce0f57))
- update envtest to match IPInRange service matching logic ([4f11a1d](https://github.com/somaz94/helios-lb/commit/4f11a1d50959e8f97f6dff5c2f9cf2e17515e57e))
- add trap cleanup and configurable IP to Helm test ([a60f0d8](https://github.com/somaz94/helios-lb/commit/a60f0d85644b660ecaa0ea3e4cd2feee59e18664))

### Documentation

- add webhook and cert-manager documentation ([4c4959a](https://github.com/somaz94/helios-lb/commit/4c4959a72572be15e055e67cfc9b08adf57c4417))
- update samples, Helm templates, and add controller tests ([8fe0cbb](https://github.com/somaz94/helios-lb/commit/8fe0cbb531d0bc9ea889e24fbb7ab504b8e69c60))
- update README, samples, and Helm CRD for new features ([9dd337c](https://github.com/somaz94/helios-lb/commit/9dd337cb5030718a65c4d6b90a7690556efa0159))
- add DEVELOPMENT.md ([c1a3c3d](https://github.com/somaz94/helios-lb/commit/c1a3c3dafb9453a69716a79d1314936937dc6cd2))
- add no-push rule to CLAUDE.md ([710cf27](https://github.com/somaz94/helios-lb/commit/710cf270e713d46ca4dda882326ea9cb4f1cb6a1))
- add CLAUDE.md project guide ([3103f67](https://github.com/somaz94/helios-lb/commit/3103f67031d847cc1e45f52443753bdd3797c6ba))
- add badges to README ([4275af8](https://github.com/somaz94/helios-lb/commit/4275af8cd3cb2fb5dffe5a58a75eb9ff51e244ce))
- unify installation structure and fix version references ([bf8a5ae](https://github.com/somaz94/helios-lb/commit/bf8a5ae205fa00bbad88b671d5b83ff694e9f985))

### Tests

- increase coverage to 90%+ for all packages ([d0eb7fe](https://github.com/somaz94/helios-lb/commit/d0eb7fe7fd4e4e7afcf3004917e472717f1a94cc))
- increase coverage to 94.7% controller, 100% network ([062e707](https://github.com/somaz94/helios-lb/commit/062e707dfeb224c3470963b4c9a143990c132983))

### Continuous Integration

- add auto-generated PR body script for make pr ([876c50f](https://github.com/somaz94/helios-lb/commit/876c50ff9bf3d356a7aa6edadd3fdbeae884f016))
- migrate gitlab-mirror workflow to multi-git-mirror action ([baebb33](https://github.com/somaz94/helios-lb/commit/baebb33d54a02932a7f1e6b231273eb23a2ab15f))
- use somaz94/contributors-action@v1 for contributors generation ([4117a20](https://github.com/somaz94/helios-lb/commit/4117a20cb23fb7bff6cceb0da431ce86ab798025))
- use major-tag-action for version tag updates ([bf17783](https://github.com/somaz94/helios-lb/commit/bf177838401c39015a692424c3f10be6cebb73f6))
- migrate changelog generator to go-changelog-action ([71d779d](https://github.com/somaz94/helios-lb/commit/71d779d19f9bfcb21084c8941654f04a629fd8b2))
- unify changelog-generator with flexible tag pattern ([9ee9847](https://github.com/somaz94/helios-lb/commit/9ee9847e7b39d48297a88682f847a9d1f86fae2a))

### Styles

- fix gofmt alignment in webhook test ([70a67b0](https://github.com/somaz94/helios-lb/commit/70a67b05e0b7ea51a3e8a04018608b0483530fdb))

### Chores

- bump version to v0.3.0 ([9c2d9f4](https://github.com/somaz94/helios-lb/commit/9c2d9f4679d28efe3c922d83cbc370bbdd42fff7))
- **deps:** bump the go-minor group with 3 updates (#37) ([#37](https://github.com/somaz94/helios-lb/pull/37)) ([e242aea](https://github.com/somaz94/helios-lb/commit/e242aea4cc67d6a1f55f2221c6f0aa2ae1d8e9af))
- add workflow Makefile targets (check-gh, branch, pr) ([07f47f6](https://github.com/somaz94/helios-lb/commit/07f47f626cd29e858a8c614aa73e4bd2f82dfb46))
- add build-time version injection and OCI labels to Dockerfile ([b777a6a](https://github.com/somaz94/helios-lb/commit/b777a6a96dd219f681d0b5dfd9cda8902190b38c))
- add version check and bump-version script ([3b43815](https://github.com/somaz94/helios-lb/commit/3b43815c09828123a3590a3bbac0304a9cd2b26f))
- change license from MIT to Apache 2.0 ([59b598b](https://github.com/somaz94/helios-lb/commit/59b598b0074b0568baf9411ddbe497aae1e5938a))

### Contributors

- somaz

<br/>

## [v0.2.6](https://github.com/somaz94/helios-lb/compare/v0.2.5...v0.2.6) (2026-03-12)

### Features

- add free IP scanner and enhance IP conflict detection ([d9b53b6](https://github.com/somaz94/helios-lb/commit/d9b53b6e246eb3f478f8e0a4a3e3454a168adff0))
- add loadBalancerClass filtering to avoid conflicts with other LB controllers ([f595ca3](https://github.com/somaz94/helios-lb/commit/f595ca3ef4d4263f21b7e96ea9e2dce7f144f8d5))
- add test-integration and test-helm scripts with Makefile targets ([6216eac](https://github.com/somaz94/helios-lb/commit/6216eaccd2dce4a40f407dcb7f720869fdd51c4a))

### Bug Fixes

- show full undeploy output in integration test cleanup ([3e24655](https://github.com/somaz94/helios-lb/commit/3e24655b79dca0635159398d5911254682aff0ce))
- add trap for cleanup and undeploy on test exit ([eced16e](https://github.com/somaz94/helios-lb/commit/eced16e59659bd6cafd820c8719cb5a13f96a3c0))
- use IP range matching instead of string comparison for service filtering ([ed5cd11](https://github.com/somaz94/helios-lb/commit/ed5cd116f5d44274373f8773588e1355e3de1d25))
- clear service ingress on HeliosConfig deletion ([b8f3e04](https://github.com/somaz94/helios-lb/commit/b8f3e04df44761655ec484f735f2e96812aafd93))
- force imagePullPolicy Always during integration tests ([96bda78](https://github.com/somaz94/helios-lb/commit/96bda7857d33463f77db43076809783190f37bb0))
- sync State and Phase fields in HeliosConfig status ([efddaa1](https://github.com/somaz94/helios-lb/commit/efddaa130a783672347542135fb43524227b872d))
- resolve critical bugs and security issues ([d4eecbf](https://github.com/somaz94/helios-lb/commit/d4eecbfb66b89b847be678fe9f9137ba3d2c4fd6))
- add pod creation wait loop before kubectl wait in integration test ([0677716](https://github.com/somaz94/helios-lb/commit/0677716639967c943cc3a6cc88ff0e70c2f15093))
- auto-install CRD and deploy controller in integration test ([08733af](https://github.com/somaz94/helios-lb/commit/08733afc9737ed3ba73e601c12273ad908053104))
- rename crd-clenaup typo, add dedicated RBAC for cleanup hook, fix Chart version ([86c71c5](https://github.com/somaz94/helios-lb/commit/86c71c540af2a8b32c9abd6b8abd6b3c1fbe7070))
- set GOTOOLCHAIN from go.mod to fix covdata errors ([e3763e0](https://github.com/somaz94/helios-lb/commit/e3763e074d2677fe282b7f7985be7d2611a28f64))
- remove helm-repo before checkout to prevent untracked file conflict ([4803add](https://github.com/somaz94/helios-lb/commit/4803add2ccc279e59313a15607944fc76da8f42e))
- resolve helm-repo checkout conflict in helm-release workflow ([ea7bf13](https://github.com/somaz94/helios-lb/commit/ea7bf13265bc68fb6f0dcda7ad341888972a1361))
- dynamically resolve tags in changelog and release workflows ([82e83d7](https://github.com/somaz94/helios-lb/commit/82e83d704d5e1babfbb477180e149b807ae55248))
- test/e2e/e2e_test.go ([a97d8ab](https://github.com/somaz94/helios-lb/commit/a97d8ab8283f5d1c41f2f789f9008ab5b1187113))
- heliosconfig_controller_test.go ([81efe26](https://github.com/somaz94/helios-lb/commit/81efe2686a6d228cdd7af4b27400a33f52fe6aa0))
- balancer.go, balancer_test.go ([6776a0e](https://github.com/somaz94/helios-lb/commit/6776a0e643b7a815038298bc474b825e8ddf2d23))
- balancer_test.go, metrics.go ([28050dd](https://github.com/somaz94/helios-lb/commit/28050ddb653b47cd6d94c379449ac97ac13a16c8))
- metrics_test.go, network_test.go, health.go ([35601ab](https://github.com/somaz94/helios-lb/commit/35601ab722b460521c3dfa1e4b4cd9d2cc507f4c))
- test, test-e2e.yml ([b622f3e](https://github.com/somaz94/helios-lb/commit/b622f3e165e7de218301c90a0ec5985bd1226a7c))
- changelog-generator.yml, .gitignore, add: gitlab-mirror.yml ([81c39e7](https://github.com/somaz94/helios-lb/commit/81c39e7b3d00045af61d2558efe1839a2b580f10))
- changelog-generator.yml ([e4ad771](https://github.com/somaz94/helios-lb/commit/e4ad77169c7efbcac7dfc4d6ccb91ed4db2b7a83))
- chagnelog-generator.yml ([47cf686](https://github.com/somaz94/helios-lb/commit/47cf68669802f256a346aa097be800b98f1ee98f))
- release.yml & changelog-generator.yml ([2afee99](https://github.com/somaz94/helios-lb/commit/2afee995a4782cd3b2ae35721354ecb1848c05f5))

### Performance Improvements

- optimize Dockerfile with build cache and smaller binary ([f3264e8](https://github.com/somaz94/helios-lb/commit/f3264e83e852e9e5ed3d0e63112d43899fab8bf8))

### Documentation

- add TESTING.md for test guide and procedures ([76b0664](https://github.com/somaz94/helios-lb/commit/76b0664a266d756ea545faddb3b37c623990702b))
- docs/VERSION_BUMP.md ([6ef8d20](https://github.com/somaz94/helios-lb/commit/6ef8d20735620fc31eddd93d3806b09675e228fc))
- add TROUBLESHOOTING.md and CONTRIBUTING.md, migrate golangci-lint to v2 ([c99731d](https://github.com/somaz94/helios-lb/commit/c99731d1c57015ef2f63f308fa11240351cac0b7))
- consolidate documentation into docs/ directory ([2cacd9f](https://github.com/somaz94/helios-lb/commit/2cacd9f5cfd3558982350bf98f39de981b8d6784))
- README.md ([644a9c0](https://github.com/somaz94/helios-lb/commit/644a9c01a4e9f3b72d5b4ec1396f507f30c5d5eb))

### Tests

- enhance loadbalancer coverage from 88.5% to 99.4% ([315a3f3](https://github.com/somaz94/helios-lb/commit/315a3f32472998d66e2c8f486adc37ccb2643b0a))
- enhance controller test coverage from 65.4% to 95.1% ([6500e33](https://github.com/somaz94/helios-lb/commit/6500e339d6fad0cb38608c7449b1bf8396a8d239))

### Continuous Integration

- add automation workflows and manifests verification ([540f8ce](https://github.com/somaz94/helios-lb/commit/540f8ce79e6e85e407c4474d85fa0fa01739bc6c))
- migrate release workflow to git-cliff and softprops/action-gh-release ([45e460e](https://github.com/somaz94/helios-lb/commit/45e460e91d7a38fa5490223a94a521cceec1ae08))
- add release notes categorization config ([2ef44a7](https://github.com/somaz94/helios-lb/commit/2ef44a73838119a094de119ababbed5abf6dd7ba))
- add helm chart release workflow for automated packaging ([cc68323](https://github.com/somaz94/helios-lb/commit/cc683232d7ce17ba2b815e21b4f100276b7f12ff))
- use conventional commit message in changelog-generator workflow ([342e1c3](https://github.com/somaz94/helios-lb/commit/342e1c3fe19c9abd416d87239b3f4d9a3a121b81))

### Chores

- add --help option to find-free-ip and HELP=1 make target ([c4c15d4](https://github.com/somaz94/helios-lb/commit/c4c15d4115ed60612e5f04b47528917db32d7f9d))
- remove hardcoded IPs from samples, add find-free-ip make target ([9f3a7a4](https://github.com/somaz94/helios-lb/commit/9f3a7a42940abc00d1554a0b3a7b997817be607d))
- bump version to v0.2.6 ([c422355](https://github.com/somaz94/helios-lb/commit/c42235543709bdc9b2082b6a64801b2301019db1))
- upgrade Go to 1.26, use go-version-file in CI workflows ([f5be28c](https://github.com/somaz94/helios-lb/commit/f5be28ca4405bc277ba8015e472480711dd2cc9f))
- **deps:** bump sigs.k8s.io/controller-runtime in the go-minor group ([d966fae](https://github.com/somaz94/helios-lb/commit/d966fae39240316f9ccbf1923f9e13b97649f2f6))
- **deps:** bump the go-minor group across 1 directory with 4 updates ([5e8a09b](https://github.com/somaz94/helios-lb/commit/5e8a09b3a39d213b81941e2166ab60e0f7101e92))
- **deps:** bump the go-minor group across 1 directory with 6 updates ([71765d8](https://github.com/somaz94/helios-lb/commit/71765d8014f3ef364ea5d676c72457743b039c8d))
- **deps:** bump golang from 1.25 to 1.26 in the docker-minor group ([9c8af01](https://github.com/somaz94/helios-lb/commit/9c8af0172e4e49d66b1140caa72b1b354445ccab))
- **deps:** bump the go-minor group with 4 updates ([46dd57a](https://github.com/somaz94/helios-lb/commit/46dd57a9710bbf942aaa2fde10d66899420e8d5b))
- **deps:** bump the go-minor group with 5 updates ([044c7de](https://github.com/somaz94/helios-lb/commit/044c7de12a69004f4ff2452cb4f21a5bf020e51d))
- test, test-e2e ([8823b1e](https://github.com/somaz94/helios-lb/commit/8823b1ea4e9d38a878c8303131779afe3be7c9d2))
- workflows ([cd3bba6](https://github.com/somaz94/helios-lb/commit/cd3bba6a56aa7860f8f3b97625bdd0460cbecbd2))
- **deps:** bump actions/checkout from 5 to 6 ([5cf8d0c](https://github.com/somaz94/helios-lb/commit/5cf8d0c44f325128c121f20b9d46c991c7e5e6aa))
- **deps:** bump the go-minor group across 1 directory with 8 updates ([f17e348](https://github.com/somaz94/helios-lb/commit/f17e348df019cf6d4af7df4bd5c0ec937fa17291))
- **deps:** bump golangci/golangci-lint-action from 8 to 9 ([fefb9b4](https://github.com/somaz94/helios-lb/commit/fefb9b4732d4225637fe822f1b48f39bbff6bae8))
- **deps:** bump actions/setup-go from 5 to 6 ([b165e6c](https://github.com/somaz94/helios-lb/commit/b165e6ce3a4e2463e255832b55f372ae0cf92b3d))
- **deps:** bump the go-minor group across 1 directory with 7 updates ([31de007](https://github.com/somaz94/helios-lb/commit/31de007b55df311e4dc828b083a9fef6f0b9ee6b))
- **deps:** bump actions/checkout from 4 to 5 ([bb59e99](https://github.com/somaz94/helios-lb/commit/bb59e9982094c07471a34e78d1197a863b4b8748))
- **deps:** bump golang from 1.24 to 1.25 in the docker-minor group ([6acea24](https://github.com/somaz94/helios-lb/commit/6acea245820ac45dbc9fedd32e54c5efaf0a512a))
- **deps:** bump golangci/golangci-lint-action from 7 to 8 ([2f55d57](https://github.com/somaz94/helios-lb/commit/2f55d573dcb5b1cf8965f738dd34dda06eb72f49))
- **deps:** bump the go-minor group with 3 updates ([6f52fe8](https://github.com/somaz94/helios-lb/commit/6f52fe8624e2d94004e1171e6839cbe07a6573bc))
- **deps:** bump the go-minor group across 1 directory with 3 updates ([3c9cc2a](https://github.com/somaz94/helios-lb/commit/3c9cc2ab2149248e3fe1ca24ed5ce43c91b0cf50))
- **deps:** bump sigs.k8s.io/controller-runtime in the go-minor group ([02850f9](https://github.com/somaz94/helios-lb/commit/02850f9fc6f410c3ef786e84a9849369379867d9))
- **deps:** bump golangci/golangci-lint-action from 6 to 7 ([4e6fe67](https://github.com/somaz94/helios-lb/commit/4e6fe67ed85edccd6babbabca99880eb183bcd3c))
- **deps:** bump the go-minor group with 2 updates ([1d36167](https://github.com/somaz94/helios-lb/commit/1d361670bb8384ff521cac425145954569f00b01))
- **deps:** bump the go-minor group with 3 updates ([9d9a466](https://github.com/somaz94/helios-lb/commit/9d9a46627ed400ab28365bca02342cf6ce0f8598))
- **deps:** bump the go-minor group with 3 updates ([69f096a](https://github.com/somaz94/helios-lb/commit/69f096aaaf9eae9d733ae8469199a5ec166dba6e))
- **deps:** bump github.com/prometheus/client_golang ([8f23d86](https://github.com/somaz94/helios-lb/commit/8f23d86284e21d2577e9d36caba47f65431d1b53))
- **deps:** bump golang from 1.23 to 1.24 in the docker-minor group ([2cc88c9](https://github.com/somaz94/helios-lb/commit/2cc88c9d3ca530f47317606dde3e005e5cd37bc0))
- **deps:** bump the go-minor group with 4 updates ([24af2d9](https://github.com/somaz94/helios-lb/commit/24af2d9a732521113b48200aebfb687d54767b1c))

### Contributors

- somaz

<br/>

## [v0.2.5](https://github.com/somaz94/helios-lb/compare/v0.2.4...v0.2.5) (2025-02-03)

### Chores

- fix release/install.yaml ([5023997](https://github.com/somaz94/helios-lb/commit/502399707d94498747304fd94a8bf70e60e2d152))
- fix workflow test test-e2e ([3fba142](https://github.com/somaz94/helios-lb/commit/3fba142e547f04e29becefc885fc0c02d84abe1c))
- upgrade go package version ([58bbe62](https://github.com/somaz94/helios-lb/commit/58bbe62d3a16641a95c3d5b2db8ae1a13802f09e))
- **deps:** bump the go-minor group with 8 updates ([3660b82](https://github.com/somaz94/helios-lb/commit/3660b82882191a2e15481737130ed557b45733bb))
- **deps:** bump janheinrichmerker/action-github-changelog-generator ([b2475a3](https://github.com/somaz94/helios-lb/commit/b2475a3293c4d045597fa0644d9cefcff43df989))
- **deps:** bump golang from 1.22 to 1.23 in the docker-minor group ([8ec9d53](https://github.com/somaz94/helios-lb/commit/8ec9d5304f9877336fc5e8bcd185e1bf06198f50))
- add dependabot.yml & fix changelog-generator.yml ([7d0059c](https://github.com/somaz94/helios-lb/commit/7d0059cade039750757d3c8093552676dd731714))
- fix changelog workflow ([2325557](https://github.com/somaz94/helios-lb/commit/2325557c794045f7a2e3d23ce67e8b0f7c88554b))
- fix changelog workflow ([fe28a3e](https://github.com/somaz94/helios-lb/commit/fe28a3e49a201e58c4b18f57f2b8ca48efe8dc76))
- fix changelog workflow ([4886d4a](https://github.com/somaz94/helios-lb/commit/4886d4ad92ab5706ef26b0d60581721f9f4cf87f))
- fix changelog workflow ([2a00d7d](https://github.com/somaz94/helios-lb/commit/2a00d7d485a2023c78e65c06137e04c3a676029d))
- fix changelog workflow ([5627d27](https://github.com/somaz94/helios-lb/commit/5627d27ee507a40061931e86d9e184bf677d3a3c))
- fix changelog-generator workflow ([9da58e1](https://github.com/somaz94/helios-lb/commit/9da58e1fda1dff661318bd3cce83c3eb000aafb5))
- fix changelog-generator workflow ([8851edd](https://github.com/somaz94/helios-lb/commit/8851edd84a69f67756bc76fedbb736cf78006d5c))
- add changelog-generator workflow ([1085f17](https://github.com/somaz94/helios-lb/commit/1085f177230738eddf48bb57edd589a23a69f0f9))
- fix annotations kr -> en ([ec1b001](https://github.com/somaz94/helios-lb/commit/ec1b0013aaf5d87e16a38642ec26a4c769564a31))

### Contributors

- somaz

<br/>

## [v0.2.4](https://github.com/somaz94/helios-lb/compare/v0.1.4...v0.2.4) (2024-12-20)

### Documentation

- helm/README.md ([8031f1b](https://github.com/somaz94/helios-lb/commit/8031f1b2e8333119a21109113f7c44eb75e21a63))
- helm NOTES.txt ([a9b7b73](https://github.com/somaz94/helios-lb/commit/a9b7b739918325d829e5ef634ff54a73aeac2bd6))

### Chores

- delete hostnetwork:true ([d27f4ec](https://github.com/somaz94/helios-lb/commit/d27f4ec9689415e4d043231b3743a637adb49812))
- add lifecycle ([21afd3e](https://github.com/somaz94/helios-lb/commit/21afd3efc00df00bef08109789de0e1496bd57bd))
- fix helm chart ([38f3112](https://github.com/somaz94/helios-lb/commit/38f3112392c2fd9bfb23c670007553ced3241a6b))
- fix helm chart ([4edb2f7](https://github.com/somaz94/helios-lb/commit/4edb2f795c96bd6335c387fa978261cc63eb1960))
- fix helm chart ([fd4ef91](https://github.com/somaz94/helios-lb/commit/fd4ef91e7d57d6bf0093b6ee1f95435785604ce0))
- delete helm repo ([f07bb74](https://github.com/somaz94/helios-lb/commit/f07bb7466876b17dd16db6c2385447aee00fb52d))
- add helm Repository ([bbb1dfa](https://github.com/somaz94/helios-lb/commit/bbb1dfa8e772d9d7b137688a6d2ae67fd8c22a11))

### Contributors

- somaz

<br/>

## [v0.1.4](https://github.com/somaz94/helios-lb/releases/tag/v0.1.4) (2024-12-18)

### Features

- typo ([4df23c9](https://github.com/somaz94/helios-lb/commit/4df23c92a142f0b486e71f1b85f8a482990656b7))
- protype commit ([3955c92](https://github.com/somaz94/helios-lb/commit/3955c928d53cc71389de6c5161614a5fb6f73612))

### Documentation

- README.md ([262c74e](https://github.com/somaz94/helios-lb/commit/262c74e082089279974ee1bf46c646496f2abfe3))
- README.md ([db19f8c](https://github.com/somaz94/helios-lb/commit/db19f8c1fe551a9e73d6b2e9483525e7fffe5ee9))
- README.md ([acace3c](https://github.com/somaz94/helios-lb/commit/acace3c077aac25559586d99cf7135046b1bc224))

### Contributors

- somaz

<br/>

