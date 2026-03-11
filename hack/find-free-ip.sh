#!/bin/bash
set -euo pipefail

# Find free (unused) IPs in a given subnet for helios-lb testing
#
# Usage:
#   ./hack/find-free-ip.sh                          # Scan 10.10.10.100-110 (default)
#   ./hack/find-free-ip.sh 192.168.1.100 192.168.1.110   # Custom range
#   ./hack/find-free-ip.sh 172.30.0.50 172.30.0.60 3     # Find 3 free IPs
#
# Environment:
#   SCAN_TIMEOUT=1    Ping timeout in seconds (default: 1)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

usage() {
  echo "Usage: $0 [START_IP] [END_IP] [COUNT]"
  echo ""
  echo "Scan a range of IPs and find unused ones for helios-lb testing."
  echo ""
  echo "Arguments:"
  echo "  START_IP    First IP to scan (default: 10.10.10.100)"
  echo "  END_IP      Last IP to scan  (default: 10.10.10.110)"
  echo "  COUNT       Number of free IPs needed (default: 2)"
  echo ""
  echo "Environment variables:"
  echo "  SCAN_TIMEOUT  Ping timeout in seconds (default: 1)"
  echo ""
  echo "Examples:"
  echo "  $0                                      # Scan 10.10.10.100-110"
  echo "  $0 192.168.1.100 192.168.1.120          # Custom range"
  echo "  $0 172.30.0.50 172.30.0.70 3            # Find 3 free IPs"
  echo ""
  echo "Via Makefile:"
  echo "  make find-free-ip"
  echo "  make find-free-ip START_IP=192.168.1.100 END_IP=192.168.1.120"
  echo "  make find-free-ip START_IP=172.30.0.50 END_IP=172.30.0.70 COUNT=3"
  exit 0
}

# Show help if requested
case "${1:-}" in
  -h|--help|help) usage ;;
esac

START_IP="${1:-10.10.10.100}"
END_IP="${2:-10.10.10.110}"
NEED_COUNT="${3:-2}"
SCAN_TIMEOUT="${SCAN_TIMEOUT:-1}"

# Convert IP to integer for range iteration
ip_to_int() {
  local IFS='.'
  read -r a b c d <<< "$1"
  echo $(( (a << 24) + (b << 16) + (c << 8) + d ))
}

# Convert integer back to IP
int_to_ip() {
  local n=$1
  echo "$(( (n >> 24) & 255 )).$(( (n >> 16) & 255 )).$(( (n >> 8) & 255 )).$(( n & 255 ))"
}

# Check if IP is in use using multiple methods
check_ip() {
  local ip=$1
  local in_use=false
  local methods=""

  # Method 1: ping (ICMP)
  if ping -c 1 -W "$SCAN_TIMEOUT" "$ip" >/dev/null 2>&1; then
    in_use=true
    methods="ping"
  fi

  # Method 2: arping (ARP layer - more reliable on L2)
  if [ "$in_use" = false ] && command -v arping >/dev/null 2>&1; then
    if arping -c 1 -w "$SCAN_TIMEOUT" "$ip" >/dev/null 2>&1; then
      in_use=true
      methods="arping"
    fi
  fi

  # Method 3: ARP cache
  if [ "$in_use" = false ]; then
    if arp -n "$ip" 2>/dev/null | grep -qv "incomplete\|no entry"; then
      in_use=true
      methods="arp-cache"
    fi
  fi

  # Method 4: K8s service LoadBalancer IP check
  if [ "$in_use" = false ] && command -v kubectl >/dev/null 2>&1; then
    if kubectl get svc -A -o jsonpath='{range .items[*]}{.status.loadBalancer.ingress[0].ip}{"\n"}{end}' 2>/dev/null | grep -q "^${ip}$"; then
      in_use=true
      methods="k8s-svc"
    fi
    # Also check spec.loadBalancerIP
    if [ "$in_use" = false ]; then
      if kubectl get svc -A -o jsonpath='{range .items[*]}{.spec.loadBalancerIP}{"\n"}{end}' 2>/dev/null | grep -q "^${ip}$"; then
        in_use=true
        methods="k8s-spec"
      fi
    fi
  fi

  if [ "$in_use" = true ]; then
    echo -e "  ${RED}✗${NC} ${ip} - in use (${methods})"
    return 0
  else
    echo -e "  ${GREEN}✓${NC} ${ip} - free"
    return 1
  fi
}

echo -e "${CYAN}=== Helios LB - Free IP Scanner ===${NC}"
echo -e "Range: ${START_IP} ~ ${END_IP}"
echo -e "Looking for ${NEED_COUNT} free IP(s)..."
echo ""

# Show detection methods
echo -e "${CYAN}Detection methods:${NC}"
echo "  1. ping (ICMP echo)"
if command -v arping >/dev/null 2>&1; then
  echo "  2. arping (ARP request) ✓"
else
  echo -e "  2. arping (ARP request) ${YELLOW}not installed${NC} - install for better L2 detection"
fi
echo "  3. arp cache lookup"
if command -v kubectl >/dev/null 2>&1; then
  echo "  4. K8s service IP check ✓"
else
  echo -e "  4. K8s service IP check ${YELLOW}kubectl not found${NC}"
fi
echo ""

echo -e "${CYAN}Scanning...${NC}"

START_INT=$(ip_to_int "$START_IP")
END_INT=$(ip_to_int "$END_IP")

FREE_IPS=()
USED_COUNT=0

for (( i=START_INT; i<=END_INT; i++ )); do
  ip=$(int_to_ip "$i")
  if ! check_ip "$ip"; then
    FREE_IPS+=("$ip")
  else
    USED_COUNT=$((USED_COUNT + 1))
  fi
done

TOTAL=$(( END_INT - START_INT + 1 ))
FREE_COUNT=${#FREE_IPS[@]}

echo ""
echo -e "${CYAN}=== Scan Result ===${NC}"
echo -e "  Total scanned: ${TOTAL}"
echo -e "  ${GREEN}Free: ${FREE_COUNT}${NC}"
echo -e "  ${RED}Used: ${USED_COUNT}${NC}"
echo ""

if [ "$FREE_COUNT" -ge "$NEED_COUNT" ]; then
  echo -e "${GREEN}Found ${FREE_COUNT} free IP(s). Suggested test IPs:${NC}"
  echo ""
  echo -e "  TEST_IP=${FREE_IPS[0]}"
  if [ "$FREE_COUNT" -ge 2 ]; then
    echo -e "  TEST_IP2=${FREE_IPS[1]}"
  fi
  if [ "$FREE_COUNT" -ge 2 ]; then
    echo -e "  TEST_IP_RANGE=${FREE_IPS[0]}-${FREE_IPS[$((FREE_COUNT - 1))]}"
  fi
  echo ""
  echo -e "${CYAN}Run integration test with:${NC}"
  if [ "$FREE_COUNT" -ge 2 ]; then
    echo -e "  TEST_IP=${FREE_IPS[0]} TEST_IP2=${FREE_IPS[1]} TEST_IP_RANGE=${FREE_IPS[0]}-${FREE_IPS[$((FREE_COUNT - 1))]} make test-integration"
  else
    echo -e "  TEST_IP=${FREE_IPS[0]} make test-integration"
  fi
else
  echo -e "${RED}Not enough free IPs (need ${NEED_COUNT}, found ${FREE_COUNT}).${NC}"
  echo -e "Try a different range:"
  echo -e "  ./hack/find-free-ip.sh 172.30.0.100 172.30.0.120"
  exit 1
fi
