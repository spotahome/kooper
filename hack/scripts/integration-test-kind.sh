#!/bin/bash

set -o errexit
set -o nounset

KUBERNETES_VERSION=v${KUBERNETES_VERSION:-1.13.6}
current_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

SUDO=''
if [[ $(id -u) -ne 0 ]]
then
    SUDO="sudo"
fi

function cleanup {
    echo "=> Removing kind cluster"
    $SUDO kind delete cluster
}
trap cleanup EXIT

echo "=> Preparing minikube for running integration tests"
$SUDO kind create cluster --image kindest/node:${KUBERNETES_VERSION}

echo "=> Waiting to start cluster..."
sleep 30

export KUBECONFIG="$(kind get kubeconfig-path)"

$SUDO chmod a+r ${KUBECONFIG}

echo "=> Running integration tests"
${current_dir}/integration-test.sh
