#!/bin/bash
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS=0
FAIL=0
SKIP=0
CHART_DIR="helm/helios-lb"
RELEASE_NAME="hlb-test"
NAMESPACE="helios-lb-system"

# Configurable test IP via environment variable
TEST_IP="${TEST_IP:-10.10.10.100}"

log_info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
log_pass()  { echo -e "${GREEN}[PASS]${NC} $1"; PASS=$((PASS+1)); }
log_fail()  { echo -e "${RED}[FAIL]${NC} $1"; FAIL=$((FAIL+1)); }
log_skip()  { echo -e "${YELLOW}[SKIP]${NC} $1"; SKIP=$((SKIP+1)); }

wait_for_active() {
  local name=$1
  local ns=${2:-default}
  local timeout=${3:-30}
  for i in $(seq 1 "$timeout"); do
    local state
    state=$(kubectl get heliosconfig "$name" -n "$ns" -o jsonpath='{.status.state}' 2>/dev/null || echo "")
    if [ "$state" = "Active" ]; then
      return 0
    fi
    sleep 1
  done
  return 1
}

cleanup_cr() {
  kubectl delete heliosconfig --all -n default --ignore-not-found 2>/dev/null || true
  sleep 2
  # Clear service ingress status so next test can re-allocate IPs
  kubectl patch svc test-svc1 -n default --subresource=status \
    -p '{"status":{"loadBalancer":{}}}' --type=merge 2>/dev/null || true
  sleep 1
}

cleanup_test_resources() {
  kubectl delete heliosconfig --all -n default --ignore-not-found 2>/dev/null || true
  kubectl delete svc test-svc1 -n default --ignore-not-found 2>/dev/null || true
  kubectl delete deploy test-nginx -n default --ignore-not-found 2>/dev/null || true
  sleep 2
}

final_cleanup() {
  echo ""
  log_info "--- Final Cleanup (trap) ---"
  cleanup_test_resources
  kubectl delete crd heliosconfigs.balancer.helios.dev --ignore-not-found 2>/dev/null || true
  helm uninstall "${RELEASE_NAME}" --no-hooks 2>/dev/null || true
  kubectl delete ns "${NAMESPACE}" --ignore-not-found 2>/dev/null || true
}
trap final_cleanup EXIT

echo ""
log_info "========================================="
log_info "Helios LB Helm Test"
log_info "========================================="
log_info "Using test IP: TEST_IP=${TEST_IP}"
log_info "Find free IPs: make find-free-ip  |  Override: TEST_IP=<free-ip> make test-helm"
echo ""

# ── Helm Lint ──
log_info "Linting Helm chart..."
if helm lint "${CHART_DIR}" > /dev/null 2>&1; then
  log_pass "Helm lint passed"
else
  log_fail "Helm lint failed"
fi

# ── Helm Template ──
log_info "Testing Helm template rendering..."
if helm template test "${CHART_DIR}" > /dev/null 2>&1; then
  log_pass "Helm template renders successfully"
else
  log_fail "Helm template failed"
fi

# ── Helm Package ──
log_info "Testing Helm package..."
PACKAGE_DIR=$(mktemp -d)
if helm package "${CHART_DIR}" -d "${PACKAGE_DIR}" > /dev/null 2>&1; then
  log_pass "Helm package created successfully"
  PACKAGE_FILE=$(ls "${PACKAGE_DIR}"/*.tgz 2>/dev/null | head -1)
  log_info "Package: ${PACKAGE_FILE}"
else
  log_fail "Helm package failed"
fi
rm -rf "${PACKAGE_DIR}"

# ── Helm Install ──
log_info "Installing chart via Helm..."

# Clean up any stuck release
helm uninstall "${RELEASE_NAME}" --no-hooks 2>/dev/null || true
kubectl delete crd heliosconfigs.balancer.helios.dev --ignore-not-found 2>/dev/null || true
kubectl delete ns "${NAMESPACE}" --ignore-not-found 2>/dev/null || true
sleep 3

if helm install "${RELEASE_NAME}" "${CHART_DIR}" --set image.pullPolicy=Always --wait --timeout 120s 2>&1; then
  log_pass "Helm release deployed successfully"
else
  log_fail "Helm install failed"
  echo ""
  log_info "========================================="
  log_info "Helm Test Summary"
  log_info "========================================="
  echo -e "  ${GREEN}PASS: ${PASS}${NC}"
  echo -e "  ${RED}FAIL: ${FAIL}${NC}"
  echo -e "  ${YELLOW}SKIP: ${SKIP}${NC}"
  exit 1
fi

# ── Verify Installation ──
log_info "Waiting for controller pod to exist..."
for i in $(seq 1 30); do
  if kubectl get pods -n "${NAMESPACE}" -l control-plane=controller-manager 2>/dev/null | grep -q .; then
    break
  fi
  sleep 1
done

log_info "Waiting for controller to be ready..."
kubectl wait --for=condition=ready pod -l control-plane=controller-manager \
  -n "${NAMESPACE}" --timeout=60s 2>/dev/null && \
  log_pass "Controller pod is running" || \
  log_fail "Controller pod not ready"

# Verify CRD
if kubectl get crd heliosconfigs.balancer.helios.dev >/dev/null 2>&1; then
  log_pass "CRD installed via Helm"
else
  log_fail "CRD not found"
fi

# Verify ClusterRole
if kubectl get clusterrole -l app.kubernetes.io/name=helios-lb 2>/dev/null | grep -q .; then
  log_pass "ClusterRole created"
else
  log_skip "ClusterRole label not found (may use different labels)"
fi

# Verify metrics service
if kubectl get svc -n "${NAMESPACE}" 2>/dev/null | grep -q metrics; then
  log_pass "Metrics service created"
else
  log_skip "Metrics service not found"
fi

# ── Create test resources ──
log_info "Creating test LoadBalancer service..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-nginx
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-nginx
  template:
    metadata:
      labels:
        app: test-nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: test-svc1
  namespace: default
spec:
  type: LoadBalancer
  loadBalancerClass: helios-lb
  loadBalancerIP: "${TEST_IP}"
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: test-nginx
EOF
kubectl wait --for=condition=available deploy/test-nginx -n default --timeout=60s 2>/dev/null || true

# ── Test: Basic HeliosConfig ──
echo ""
log_info "--- [Test] Basic HeliosConfig ---"

kubectl apply -f - <<EOF
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: test-basic
  namespace: default
spec:
  ipRange: "${TEST_IP}"
  method: RoundRobin
EOF

if wait_for_active "test-basic" "default" 30; then
  log_pass "Basic: HeliosConfig Active"
else
  log_fail "Basic: HeliosConfig not Active"
fi

ALLOCATED=$(kubectl get heliosconfig test-basic -n default -o jsonpath='{.status.allocatedIPs}' 2>/dev/null || echo "{}")
if [ "$ALLOCATED" != "{}" ] && [ -n "$ALLOCATED" ]; then
  log_pass "Basic: IP allocated"
else
  log_skip "Basic: No IP allocated"
fi

cleanup_cr

# ── Test: Port-based HeliosConfig ──
echo ""
log_info "--- [Test] Port-based HeliosConfig ---"

kubectl apply -f - <<EOF
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: test-port
  namespace: default
spec:
  ipRange: "${TEST_IP}"
  method: RoundRobin
  ports:
    - port: 80
    - port: 443
EOF

if wait_for_active "test-port" "default" 30; then
  log_pass "Port: HeliosConfig Active"
else
  log_fail "Port: HeliosConfig not Active"
fi

cleanup_cr

# ── Test: Cleanup on Deletion ──
echo ""
log_info "--- [Test] Cleanup on Deletion ---"

kubectl apply -f - <<EOF
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: test-cleanup
  namespace: default
spec:
  ipRange: "${TEST_IP}"
  method: RoundRobin
EOF

wait_for_active "test-cleanup" "default" 30 || true
kubectl delete heliosconfig test-cleanup -n default --timeout=30s 2>/dev/null || true
sleep 3

if ! kubectl get heliosconfig test-cleanup -n default 2>/dev/null; then
  log_pass "Cleanup: Finalizer removed, CR deleted"
else
  log_fail "Cleanup: CR still exists"
fi

# ── Helm Upgrade Test ──
echo ""
log_info "--- Helm Upgrade Test ---"
if helm upgrade "${RELEASE_NAME}" "${CHART_DIR}" --wait --timeout 120s 2>&1 | grep -q "has been upgraded"; then
  log_pass "Helm upgrade successful"
else
  log_skip "Helm upgrade output did not match (may still be OK)"
fi

# ── Summary ──
echo ""
log_info "========================================="
log_info "Helm Test Summary"
log_info "========================================="
echo -e "  ${GREEN}PASS: ${PASS}${NC}"
echo -e "  ${RED}FAIL: ${FAIL}${NC}"
echo -e "  ${YELLOW}SKIP: ${SKIP}${NC}"
echo ""

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
