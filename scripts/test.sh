#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

CGO_ENABLED=0 go test -v -cover --count=1 ./...
