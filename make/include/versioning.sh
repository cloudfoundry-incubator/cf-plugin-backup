#!/bin/sh

set +o errexit +o nounset

test -n "${XTRACE}" && set -o xtrace

set -o errexit -o nounset

GIT_ROOT=${GIT_ROOT:-$(git rev-parse --show-toplevel)}
GIT_DESCRIBE=${GIT_DESCRIBE:-$(git describe --always --tags --long)}
GIT_BRANCH=${GIT_BRANCH:-$(git name-rev --name-only HEAD)}

GIT_TAG=${GIT_TAG:-$(echo ${GIT_DESCRIBE} | awk -F - '{ print $1 }' )}
GIT_COMMITS=${GIT_COMMITS:-$(echo ${GIT_DESCRIBE} | awk -F - '{ print $2 }' )}
GIT_SHA=${GIT_SHA:-$(echo ${GIT_DESCRIBE} | awk -F - '{ print $3 }' )}

ARTIFACT_NAME=${ARTIFACT_NAME:-$(basename $(git config --get remote.origin.url) .git | sed s/^hcf-//)}
#if we don't have a release don't add GIT_COMMITS and GIT_SHA in the version
if [ -z "${GIT_COMMITS}" ] && [ -z "${GIT_SHA}" ] 
    then
        ARTIFACT_VERSION="1.0.0"+${GIT_TAG}.${GIT_BRANCH}
    else
        ARTIFACT_VERSION=${GIT_TAG}+${GIT_COMMITS}.${GIT_SHA}.${GIT_BRANCH}
fi

APP_VERSION=${ARTIFACT_NAME}-${ARTIFACT_VERSION}

set +o errexit +o nounset +o xtrace
