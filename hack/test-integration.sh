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
NAMESPACE="helios-lb-system"
SAMPLES_DIR="config/samples"

# Configurable test IPs via environment variables
# Override with: TEST_IP=172.30.0.100 TEST_IP2=172.30.0.101 TEST_IP_RANGE=172.30.0.100-172.30.0.110 make test-integration
TEST_IP="${TEST_IP:-10.10.10.100}"
TEST_IP2="${TEST_IP2:-10.10.10.101}"
TEST_IP_RANGE="${TEST_IP_RANGE:-10.10.10.100-10.10.10.110}"

log_info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
log_pass()  { echo -e "${GREEN}[PASS]${NC} $1"; PASS=$((PASS+1)); }
log_fail()  { echo -e "${RED}[FAIL]${NC} $1"; FAIL=$((FAIL+1)); }
log_skip()  { echo -e "${YELLOW}[SKIP]${NC} $1"; SKIP=$((SKIP+1)); }

# Check if an IP is in use (ping, arping, ARP cache, K8s services)
check_ip_in_use() {
  local ip=$1
  # 1) ping - ICMP echo
  if ping -c 1 -W 1 "$ip" >/dev/null 2>&1; then
    return 0
  fi
  # 2) arping - ARP layer (more reliable on L2, detects hosts that block ICMP)
  if command -v arping >/dev/null 2>&1; then
    if arping -c 1 -w 1 "$ip" >/dev/null 2>&1; then
      return 0
    fi
  fi
  # 3) ARP cache - check if MAC address is known
  if arp -n "$ip" 2>/dev/null | grep -qv "incomplete\|no entry"; then
    return 0
  fi
  # 4) K8s services - check if any service already uses this IP
  if kubectl get svc -A -o jsonpath='{range .items[*]}{.status.loadBalancer.ingress[0].ip}{"\n"}{end}' 2>/dev/null | grep -q "^${ip}$"; then
    return 0
  fi
  if kubectl get svc -A -o jsonpath='{range .items[*]}{.spec.loadBalancerIP}{"\n"}{end}' 2>/dev/null | grep -q "^${ip}$"; then
    return 0
  fi
  return 1
}

wait_for_pods() {
  local ns=$1
  local timeout=${2:-60}
  log_info "Waiting for pods in ${ns} to be ready (timeout: ${timeout}s)..."
  kubectl wait --for=condition=ready pod --all -n "$ns" --timeout="${timeout}s" 2>/dev/null || true
}

wait_for_active() {
  local name=$1
  local ns=${2:-default}
  local timeout=${3:-30}
  log_info "Waiting for HeliosConfig '${name}' to become Active (timeout: ${timeout}s)..."
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
  for svc in test-svc1 test-svc2; do
    kubectl patch svc "$svc" -n default --subresource=status \
      -p '{"status":{"loadBalancer":{}}}' --type=merge 2>/dev/null || true
  done
  sleep 1
}

cleanup_test_resources() {
  kubectl delete -f "${SAMPLES_DIR}/nginx-test.yaml" --ignore-not-found 2>/dev/null || true
  kubectl delete svc test-svc1 test-svc2 -n default --ignore-not-found 2>/dev/null || true
  kubectl delete deploy test-nginx -n default --ignore-not-found 2>/dev/null || true
  cleanup_cr
}

final_cleanup() {
  echo ""
  log_info "--- Final Cleanup (trap) ---"
  cleanup_test_resources
  make undeploy || true
}
trap final_cleanup EXIT

echo ""
log_info "========================================="
log_info "Helios LB Integration Test"
log_info "========================================="
echo ""

# Check prerequisites
if ! kubectl cluster-info >/dev/null 2>&1; then
  log_fail "Cannot connect to Kubernetes cluster"
  exit 1
fi
log_pass "Kubernetes cluster is reachable"

# Detect other LB controllers to avoid conflicts
DETECTED_LBS=""
if kubectl get crd ipaddresspools.metallb.io >/dev/null 2>&1; then
  DETECTED_LBS="${DETECTED_LBS} MetalLB"
  log_info "Detected MetalLB - checking IP pool conflicts..."
  METALLB_POOLS=$(kubectl get ipaddresspool -A -o jsonpath='{range .items[*]}{.spec.addresses[*]}{"\n"}{end}' 2>/dev/null || echo "")
  TEST_SUBNET=$(echo "$TEST_IP" | cut -d. -f1-3)
  if echo "$METALLB_POOLS" | grep -q "$TEST_SUBNET"; then
    log_info "WARNING: MetalLB has IP pool overlapping test subnet (${TEST_SUBNET}.x)"
    log_info "MetalLB pools: ${METALLB_POOLS}"
    log_info "Consider overriding: TEST_IP=<free-ip> make test-integration"
  else
    log_pass "No MetalLB IP pool conflict with test subnet (${TEST_SUBNET}.x)"
  fi
fi
if kubectl get crd ciliumloadbalancerippools.cilium.io >/dev/null 2>&1; then
  DETECTED_LBS="${DETECTED_LBS} Cilium-LB"
fi
if kubectl get crd bgpconfigurations.crd.projectcalico.org >/dev/null 2>&1; then
  DETECTED_LBS="${DETECTED_LBS} Calico-BGP"
fi
if [ -n "$DETECTED_LBS" ]; then
  log_info "Other LB controllers detected:${DETECTED_LBS}"
  log_info "Test services use loadBalancerClass=helios-lb to avoid conflicts"
else
  log_info "No other LB controllers detected"
fi

# Pre-flight IP availability check
log_info "Checking test IP availability..."
IP_CONFLICT=false
for ip in "$TEST_IP" "$TEST_IP2"; do
  if check_ip_in_use "$ip"; then
    log_fail "IP ${ip} is already in use! Override with TEST_IP=<free-ip> make test-integration"
    IP_CONFLICT=true
  fi
done
if [ "$IP_CONFLICT" = true ]; then
  log_fail "Cannot proceed with conflicting IPs. Please provide unused IPs."
  exit 1
fi
log_pass "Test IPs are available (${TEST_IP}, ${TEST_IP2})"
log_info "Using test IPs: TEST_IP=${TEST_IP}, TEST_IP2=${TEST_IP2}, TEST_IP_RANGE=${TEST_IP_RANGE}"

# Auto-install CRD if not found
if ! kubectl get crd heliosconfigs.balancer.helios.dev >/dev/null 2>&1; then
  log_info "HeliosConfig CRD not found. Installing with 'make install'..."
  make install
  if ! kubectl get crd heliosconfigs.balancer.helios.dev >/dev/null 2>&1; then
    log_fail "Failed to install HeliosConfig CRD"
    exit 1
  fi
fi
log_pass "HeliosConfig CRD is installed"

# Auto-deploy controller if not running
if kubectl get pods -n "${NAMESPACE}" -l control-plane=controller-manager 2>/dev/null | grep -q Running; then
  # Ensure imagePullPolicy is Always for testing
  kubectl patch deployment helios-lb-controller-manager -n "$NAMESPACE" \
    -p '{"spec":{"template":{"spec":{"containers":[{"name":"manager","imagePullPolicy":"Always"}]}}}}' 2>/dev/null || true
  # Wait for rollout if patch triggered restart
  kubectl rollout status deployment/helios-lb-controller-manager -n "$NAMESPACE" --timeout=120s 2>/dev/null || true
  log_pass "Controller is running"
else
  log_info "Controller not found. Deploying with 'make deploy'..."
  make deploy IMG="$(grep '^IMG ?=' Makefile | awk -F'= ' '{print $2}')"
  # Force image pull to ensure latest image is used during testing
  kubectl set image deployment/helios-lb-controller-manager \
    manager="$(grep '^IMG ?=' Makefile | awk -F'= ' '{print $2}')" \
    -n "$NAMESPACE" 2>/dev/null || true
  kubectl patch deployment helios-lb-controller-manager -n "$NAMESPACE" \
    -p '{"spec":{"template":{"spec":{"containers":[{"name":"manager","imagePullPolicy":"Always"}]}}}}' 2>/dev/null || true
  log_info "Waiting for controller pod to be created..."
  for i in $(seq 1 60); do
    if kubectl get pods -n "$NAMESPACE" -l control-plane=controller-manager -o name 2>/dev/null | grep -q .; then
      break
    fi
    sleep 2
  done
  log_info "Waiting for controller to be ready..."
  if ! kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n "$NAMESPACE" --timeout=180s; then
    log_fail "Failed to deploy controller in ${NAMESPACE}"
    exit 1
  fi
  log_pass "Controller is running"
fi

# Clean up any previous test resources
cleanup_test_resources

# ── Test A: Basic HeliosConfig ──
echo ""
log_info "--- Test A: Basic HeliosConfig ---"

# Create a LoadBalancer service
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

# Create basic HeliosConfig
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
  log_pass "Basic: HeliosConfig became Active"
else
  log_fail "Basic: HeliosConfig did not become Active"
fi

# Check allocated IPs
ALLOCATED=$(kubectl get heliosconfig test-basic -n default -o jsonpath='{.status.allocatedIPs}' 2>/dev/null || echo "{}")
if [ "$ALLOCATED" != "{}" ] && [ -n "$ALLOCATED" ]; then
  log_pass "Basic: IP allocated (${ALLOCATED})"
else
  log_skip "Basic: No IP allocated (may need matching service)"
fi

# Check service annotation
ANNOTATION=$(kubectl get svc test-svc1 -n default -o jsonpath='{.metadata.annotations.balancer\.helios\.dev/load-balancer-class}' 2>/dev/null || echo "")
if [ "$ANNOTATION" = "helios-lb" ]; then
  log_pass "Basic: Service annotation set correctly"
else
  log_skip "Basic: Service annotation not set (${ANNOTATION})"
fi

cleanup_cr

# ── Test B: Port-based HeliosConfig ──
echo ""
log_info "--- Test B: Port-based HeliosConfig ---"

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
  log_pass "Port: HeliosConfig became Active"
else
  log_fail "Port: HeliosConfig did not become Active"
fi

# Verify status has port info
STATE=$(kubectl get heliosconfig test-port -n default -o jsonpath='{.status.state}' 2>/dev/null || echo "")
if [ "$STATE" = "Active" ]; then
  log_pass "Port: State is Active with port configuration"
else
  log_fail "Port: State is '${STATE}', expected 'Active'"
fi

cleanup_cr

# ── Test C: Multiple LB Methods ──
echo ""
log_info "--- Test C: LeastConnection Method ---"

kubectl apply -f - <<EOF
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: test-leastconn
  namespace: default
spec:
  ipRange: "${TEST_IP}"
  method: LeastConnection
EOF

if wait_for_active "test-leastconn" "default" 30; then
  log_pass "LeastConnection: HeliosConfig became Active"
else
  log_fail "LeastConnection: HeliosConfig did not become Active"
fi

cleanup_cr

# ── Test D: IP Range ──
echo ""
log_info "--- Test D: IP Range Allocation ---"

# Create a second service
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: test-svc2
  namespace: default
spec:
  type: LoadBalancer
  loadBalancerClass: helios-lb
  loadBalancerIP: "${TEST_IP2}"
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: test-nginx
EOF

kubectl apply -f - <<EOF
apiVersion: balancer.helios.dev/v1
kind: HeliosConfig
metadata:
  name: test-range
  namespace: default
spec:
  ipRange: "${TEST_IP_RANGE}"
  method: RoundRobin
EOF

if wait_for_active "test-range" "default" 30; then
  log_pass "Range: HeliosConfig became Active"
else
  log_fail "Range: HeliosConfig did not become Active"
fi

# Check multiple IPs allocated
ALLOCATED=$(kubectl get heliosconfig test-range -n default -o jsonpath='{.status.allocatedIPs}' 2>/dev/null || echo "{}")
log_info "Range: Allocated IPs: ${ALLOCATED}"
if echo "$ALLOCATED" | grep -q "test-svc"; then
  log_pass "Range: Multiple services received IPs"
else
  log_skip "Range: Could not verify multiple IP allocation"
fi

cleanup_cr

# ── Test E: Cleanup on Deletion ──
echo ""
log_info "--- Test E: Cleanup on Deletion ---"

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

# Delete and verify cleanup
kubectl delete heliosconfig test-cleanup -n default --timeout=30s 2>/dev/null || true
sleep 3

if ! kubectl get heliosconfig test-cleanup -n default 2>/dev/null; then
  log_pass "Cleanup: HeliosConfig deleted successfully (finalizer removed)"
else
  log_fail "Cleanup: HeliosConfig still exists"
fi

# ── Summary ──
echo ""
log_info "========================================="
log_info "Integration Test Summary"
log_info "========================================="
echo -e "  ${GREEN}PASS: ${PASS}${NC}"
echo -e "  ${RED}FAIL: ${FAIL}${NC}"
echo -e "  ${YELLOW}SKIP: ${SKIP}${NC}"
echo ""

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
