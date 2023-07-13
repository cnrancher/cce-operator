#!/usr/bin/env bash

set -euo pipefail

# Ensure git installed
type git > /dev/null

DIRTY=""
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
    DIRTY="-dirty"
fi

COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=${DRONE_TAG:-$(git tag -l --contains HEAD | head -n 1)}

if [[ -z "$DIRTY" && -n "$GIT_TAG" ]]; then
    VERSION=$GIT_TAG
else
    VERSION="${COMMIT}${DIRTY}"
fi

if [[ -z "${ARCH:-}" ]]; then
    ARCH=$(go env GOHOSTARCH)
fi
if [[ -z "${OS:-}" ]]; then
    OS=$(go env GOHOSTOS)
fi

SUFFIX="-${OS}-${ARCH}"
HELM_TAG=${TAG:-${VERSION}}
HELM_VERSION=${HELM_TAG/v/}
TAG=${TAG:-${VERSION}${SUFFIX}}
REPO=${REPO:-cnrancher}

if [[ $TAG = *dirty* ]]; then
    TAG=dev
    HELM_TAG=dev
    HELM_VERSION=0.0.0-dev
    DEBUG="true"
fi

if [[ ${VERSION} == *rc* ]] || [[ ${VERSION} == *alpha* ]] || [[ ${VERSION} == *beta* ]]; then
    DEBUG="true"
fi
