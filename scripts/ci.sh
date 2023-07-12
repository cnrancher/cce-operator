#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)

./validate.sh
./test.sh
./build.sh
./package.sh
./package-helm.sh
