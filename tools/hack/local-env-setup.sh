#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

function help()
{
  cat  <<EOF
The higress local environment setup
Usage: local-env-setup.sh <[Options]>
Options:
         -h   --help   help for setup
         -c   --crd    setup istio CRD
EOF
}

IS_INSTALLED_CRD=false
while [ $# -gt 0 ]
do
    case $1 in
    -h|--help) help ; exit 1;;
    -c|--crd) IS_INSTALLED_CRD=true ;;
    (-*) echo "$0: error - unrecognized option $1" 1>&2; help; exit 1;;
    (*) break;;
    esac
    shift
done

HIGRESS_KUBECONFIG="${HOME}/.kube/config_higress"
HIGRESS_CLUSTER_NAME="higress"

echo "Step1: Create local cluster: " ${HIGRESS_KUBECONFIG}
kind delete cluster --name="${HIGRESS_CLUSTER_NAME}" 2>&1
kind create cluster --kubeconfig "${HIGRESS_KUBECONFIG}" --name "${HIGRESS_CLUSTER_NAME}"  --config=`pwd`/cluster.conf  --image kindest/node:v1.21.1
export KUBECONFIG="${HIGRESS_KUBECONFIG}"
echo "Step1: Create local cluster finished."

if [ "$IS_INSTALLED_CRD" = true ]; then
  echo " "
  echo "Step2: Installing istio CRD "
  helm repo add istio https://istio-release.storage.googleapis.com/charts
  helm install istio-base istio/base -n istio-system --create-namespace

  echo " "
  echo "Step3: Installing Higress "
  helm repo add higress.io https://higress.io/helm-charts
  helm install higress -n higress-system higress.io/higress --devel --create-namespace --render-subchart-notes  --set global.enableIstioAPI=true --set global.kind=true --set higress-console.o11y.enabled=true  --set higress-controller.domain=console.higress.io --set higress-console.admin.password.value=admin
  echo "Step3: Installing Higress finished."
else
  echo " "
  echo "Step2: Installing Higress "
  helm repo add higress.io https://higress.io/helm-charts
  helm install higress -n higress-system higress.io/higress --devel --create-namespace --render-subchart-notes  --set global.kind=true --set higress-console.o11y.enabled=true  --set higress-controller.domain=console.higress.io --set higress-console.admin.password.value=admin
  echo "Step2: Installing Higress finished."
fi

kubectl get deploy -n higress-system

echo "After all pods ready, Get the Higress Dashboard URL to visit by running these commands in the same shell:"
echo "    export KUBECONFIG=${HOME}/.kube/config_higress"
echo "    echo \"127.0.0.1 console.higress.io\" | sudo tee -a /etc/hosts"
echo "    kubectl -n higress-system port-forward service/higress-gateway 8080:80"
echo "    "
echo "    "
echo "    higress console url: http://console.higress.io:8080 , login with admin/admin"
echo "    higress grafana url: http://console.higress.io:8080/grafana"
echo "    higress prometheus url: http://console.higress.io:8080/prometheus"

