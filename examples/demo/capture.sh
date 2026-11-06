#!/usr/bin/env bash
# Runs the README demo for real on a throwaway kind cluster, with helios-lb
# installed from helm/helios-lb and its published image, and saves each command's
# output under examples/demo/captured/, which replay.sh prints back.
# Usage: examples/demo/capture.sh   (needs kind, kubectl and helm; pulls the image)
set -euo pipefail

DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "${DEMO_DIR}/../.." && pwd)"
OUT="${DEMO_DIR}/captured"
CLUSTER="helios-lb-demo"

for tool in kind kubectl helm; do
    command -v "${tool}" >/dev/null 2>&1 || { echo "${tool} not found" >&2; exit 1; }
done
if kind get clusters 2>/dev/null | grep -qx "${CLUSTER}"; then
    echo "kind cluster ${CLUSTER} already exists; delete it first" >&2
    exit 1
fi

WORK="$(mktemp -d)"
# kind would otherwise switch the current context of the user's own kubeconfig.
export KUBECONFIG="${WORK}/kubeconfig"
cleanup() {
    kind delete cluster --name "${CLUSTER}" --kubeconfig "${KUBECONFIG}" >/dev/null 2>&1 || true
    rm -rf "${WORK:?}"
}
trap cleanup EXIT

kind create cluster --name "${CLUSTER}" --kubeconfig "${KUBECONFIG}" --wait 60s
# The chart creates helios-lb-system itself, so the release record stays in default.
helm install helios-lb "${ROOT_DIR}/helm/helios-lb" --wait --timeout 3m

rm -rf "${OUT}"
mkdir -p "${OUT}"
cd "${DEMO_DIR}"
kubectl apply -f service.yaml > "${OUT}/01-kubectl" 2>&1
kubectl get svc web > "${OUT}/02-kubectl" 2>&1
kubectl apply -f heliosconfig.yaml > "${OUT}/03-kubectl" 2>&1
# Off screen: wait for the address so the next capture shows the result, not a race.
for _ in $(seq 1 60); do
    [ -n "$(kubectl get svc web -o jsonpath='{.status.loadBalancer.ingress[0].ip}')" ] && break
    sleep 1
done
kubectl get svc web > "${OUT}/04-kubectl" 2>&1
kubectl get heliosconfig > "${OUT}/05-kubectl" 2>&1

echo "captured into ${OUT}"
