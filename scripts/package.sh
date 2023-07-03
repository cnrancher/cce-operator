#!/usr/bin/env bash

set -euo pipefail

source $(dirname $0)/version.sh
cd $(dirname $0)/..
WORKINGDIR=$(pwd)

mkdir -p dist/artifacts
cp bin/cce-operator dist/artifacts/cce-operator${SUFFIX}

./scripts/package-helm.sh
