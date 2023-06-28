#!/bin/bash

set -euo pipefail

cd $(dirname $0)

./validate.sh
./build.sh
