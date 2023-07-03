#!/usr/bin/env bash

set -euo pipefail

if ! hash helm 2>/dev/null; then
    exit 0
fi

cd $(dirname $0)/..
WORKINGDIR=$(pwd)
source ./scripts/version.sh

rm -rf build/charts &> /dev/null || echo -n ""
mkdir -p build dist/artifacts &> /dev/null || echo -n ""
cp -rf charts build/ &> /dev/null || echo -n ""

sed -i \
    -e 's/^version:.*/version: '${HELM_VERSION}'/' \
    -e 's/appVersion:.*/appVersion: '${HELM_VERSION}'/' \
    build/charts/cce-operator/Chart.yaml

sed -i \
    -e 's/tag:.*/tag: '${HELM_TAG}'/' \
    build/charts/cce-operator/values.yaml

sed -i \
    -e 's/^version:.*/version: '${HELM_VERSION}'/' \
    -e 's/appVersion:.*/appVersion: '${HELM_VERSION}'/' \
    build/charts/cce-operator-crd/Chart.yaml

helm package -d ./dist/artifacts ./build/charts/cce-operator
helm package -d ./dist/artifacts ./build/charts/cce-operator-crd
